package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// URL validation is repeated at delivery. DNS is resolved inside DialContext;
// validated numeric addresses are dialled directly, closing the re-resolution gap.
func validateWebhookURL(raw string, policy NotifyConfig) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return errors.New("webhook requires an absolute URL without credentials or fragment")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && policy.WebhookAllowHTTP) {
		return errors.New("webhook requires HTTPS; HTTP needs an explicit insecure exception")
	}
	if strings.Contains(u.Hostname(), "%") {
		return errors.New("webhook scoped addresses are not allowed")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
		return errors.New("webhook destination is not allowed; internal hooks need an explicit private-network exception")
	}
	return nil
}

// validateWebhookProxyURL accepts an optional http(s) egress proxy address.
// Credentials are refused so the value is never a secret.
func validateWebhookProxyURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("notify.webhook_proxy_url must be http:// or https:// host[:port] without credentials or path")
	}
	return nil
}

// checkWebhookDestination applies the destination address policy by resolving
// the webhook host. It is used when a proxy dials on our behalf; the proxy
// resolves the name again, so this narrows but cannot fully close that gap.
func checkWebhookDestination(ctx context.Context, host string, policy NotifyConfig, lookup func(context.Context, string, string) ([]netip.Addr, error)) error {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
			return errors.New("webhook destination is not allowed")
		}
		return nil
	}
	ips, err := lookup(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return errors.New("webhook DNS resolution failed")
	}
	for _, ip := range ips {
		if !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
			return errors.New("webhook DNS resolved to a forbidden address")
		}
	}
	return nil
}

type webhookDestinationGuard struct {
	next   http.RoundTripper
	policy NotifyConfig
	lookup func(context.Context, string, string) ([]netip.Addr, error)
}

func (g webhookDestinationGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := checkWebhookDestination(req.Context(), req.URL.Hostname(), g.policy, g.lookup); err != nil {
		return nil, err
	}
	return g.next.RoundTrip(req)
}

func webhookIPAllowed(ip netip.Addr, allowPrivate bool) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.Zone() != "" || !ip.IsGlobalUnicast() && !ip.IsLoopback() {
		return false
	}
	// Metadata, link-local, multicast and unspecified addresses are never enabled.
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	if ip.IsPrivate() || ip.IsLoopback() {
		return allowPrivate
	}
	for _, raw := range []string{"100.64.0.0/10", "168.63.129.16/32", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "::/96", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64"} {
		if netip.MustParsePrefix(raw).Contains(ip) {
			return false
		}
	}
	return true
}

func webhookClient(policy NotifyConfig) (*http.Client, *http.Transport) {
	var dialer net.Dialer
	return webhookClientWithNetwork(policy, net.DefaultResolver.LookupNetIP, dialer.DialContext)
}

func webhookClientWithNetwork(policy NotifyConfig, lookup func(context.Context, string, string) ([]netip.Addr, error), dial func(context.Context, string, string) (net.Conn, error)) (*http.Client, *http.Transport) {
	transport := &http.Transport{
		Proxy:                 nil, // Do not let environment proxies resolve/bypass destination policy.
		TLSHandshakeTimeout:   4 * time.Second,
		ResponseHeaderTimeout: 4 * time.Second,
		DisableKeepAlives:     true,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("invalid webhook destination")
		}
		ips, err := lookup(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("webhook DNS resolution failed")
		}
		for _, ip := range ips {
			if !webhookIPAllowed(ip, policy.WebhookAllowPrivate) {
				return nil, errors.New("webhook DNS resolved to a forbidden address")
			}
		}
		for _, ip := range ips {
			conn, dialErr := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
		}
		return nil, errors.New("webhook connection failed")
	}
	var roundTripper http.RoundTripper = transport
	if proxy, err := url.Parse(policy.WebhookProxyURL); policy.WebhookProxyURL != "" && err == nil {
		// Through a proxy we dial only the operator-chosen proxy (which may be
		// on a private network) and check the destination before each send.
		transport.Proxy = http.ProxyURL(proxy)
		transport.DialContext = dial
		roundTripper = webhookDestinationGuard{next: transport, policy: policy, lookup: lookup}
	}
	client := &http.Client{Timeout: 6 * time.Second, Transport: roundTripper,
		// Never forward alert bodies or credentials to a redirected destination.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return client, transport
}
