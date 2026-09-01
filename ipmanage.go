package main

// Optional management of site listen IPs on the host interface.
//
// When a site sets manage_ip=true and its listen address is a concrete IP that
// isn't currently on any interface, the WAF assigns it to the NIC whose subnet
// contains it (ip addr add <ip>/<prefix> dev <iface>). It removes an address
// only if it was the one that added it, and re-asserts wanted addresses on every
// apply — including at boot — so they survive reboots without Netplan edits.
//
// This needs CAP_NET_ADMIN (granted in the systemd unit) and the iproute2 `ip`
// binary (present on Debian). It is off unless a site opts in, because a real
// floating VIP should be owned by keepalived/VRRP, not pinned here.
//
// Uses only stdlib (net, os/exec); the `ip` binary is a system tool, like the
// system ssh used elsewhere — no Go library dependencies.

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type ipManager struct {
	mu    sync.Mutex
	added map[string]string // ip -> "iface/prefix" we assigned (so we only remove ours)
	log   *slog.Logger
	managementIP net.IP
	dataInterface string
}

func newIPManager(log *slog.Logger, adminAddr, dataInterface string) *ipManager {
	host, _, err := net.SplitHostPort(adminAddr)
	if err != nil {
		host = adminAddr
	}
	return &ipManager{
		added: map[string]string{}, log: log,
		managementIP: net.ParseIP(strings.Trim(host, "[]")),
		dataInterface: strings.TrimSpace(dataInterface),
	}
}

// listenIP extracts a concrete IP from a listen address, or "" for wildcard
// binds (":443", "0.0.0.0:443", "[::]:443") which need no interface assignment.
func listenIP(listen string) string {
	host := listen
	if strings.HasPrefix(listen, "[") { // [v6]:port
		if i := strings.Index(listen, "]"); i > 0 {
			host = listen[1:i]
		}
	} else if strings.Count(listen, ":") == 1 { // host:port
		host = listen[:strings.LastIndex(listen, ":")]
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return ""
	}
	if net.ParseIP(host) == nil {
		return "" // a hostname, not an IP — can't assign
	}
	return host
}

// hostHasIP reports whether ip is already on some local interface.
func hostHasIP(ip string) bool {
	target := net.ParseIP(ip)
	if target == nil {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(target) {
			return true
		}
	}
	return false
}

func interfaceWithIP(ip string) string {
	target := net.ParseIP(ip)
	if target == nil {
		return ""
	}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, addr := range addrs {
			if ipn, yes := addr.(*net.IPNet); yes && ipn.IP.Equal(target) {
				return ifc.Name
			}
		}
	}
	return ""
}

// ifaceForIP finds the interface whose subnet contains ip, returning the
// interface name and the prefix length to use for the new address.
func ifaceForIP(ip string) (iface string, prefixLen int, ok bool) {
	target := net.ParseIP(ip)
	if target == nil {
		return "", 0, false
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", 0, false
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || ifc.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil && target.To4() != nil {
				continue
			}
			if ipn.Contains(target) {
				ones, _ := ipn.Mask.Size()
				return ifc.Name, ones, true
			}
		}
	}
	return "", 0, false
}

// ifaceByName validates an explicit interface and returns a useful default
// prefix from one of its addresses of the same family as target.
func ifaceByName(name, target string) (prefixLen int, ok bool, err error) {
	ifc, err := net.InterfaceByName(strings.TrimSpace(name))
	if err != nil {
		return 0, false, fmt.Errorf("interface %q not found", name)
	}
	if ifc.Flags&net.FlagLoopback != 0 {
		return 0, false, fmt.Errorf("interface %q is loopback", name)
	}
	if ifc.Flags&net.FlagUp == 0 {
		return 0, false, fmt.Errorf("interface %q is down", name)
	}
	targetIP := net.ParseIP(target)
	addrs, err := ifc.Addrs()
	if err != nil {
		return 0, false, fmt.Errorf("read interface %q addresses: %w", name, err)
	}
	for _, addr := range addrs {
		ipn, yes := addr.(*net.IPNet)
		if !yes || (ipn.IP.To4() != nil) != (targetIP.To4() != nil) {
			continue
		}
		ones, _ := ipn.Mask.Size()
		return ones, true, nil
	}
	return 0, false, nil
}

func interfaceHasIP(name string, target net.IP) bool {
	if target == nil {
		return false
	}
	ifc, err := net.InterfaceByName(name)
	if err != nil {
		return false
	}
	addrs, _ := ifc.Addrs()
	for _, addr := range addrs {
		if ipn, yes := addr.(*net.IPNet); yes && ipn.IP.Equal(target) {
			return true
		}
	}
	return false
}

// reconcile makes host IPs match the set of sites that want managed IPs.
func (m *ipManager) reconcile(cfg Config) error {
	want := map[string]SiteConfig{}
	for _, s := range cfg.Sites {
		if !s.ManageIP {
			continue
		}
		if ip := listenIP(s.Listen); ip != "" {
			want[ip] = s
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// add wanted IPs not already present
	for ip, site := range want {
		selectedInterface := strings.TrimSpace(site.ManageInterface)
		if selectedInterface == "" {
			selectedInterface = m.dataInterface
		}
		if selectedInterface != "" && interfaceHasIP(selectedInterface, m.managementIP) {
			return fmt.Errorf("site %q manage_ip: interface %q is the management plane", site.Name, selectedInterface)
		}
		if hostHasIP(ip) {
			if selectedInterface != "" {
				wantIface := selectedInterface
				gotIface := interfaceWithIP(ip)
				if gotIface != wantIface {
					return fmt.Errorf("site %q manage_ip: %s is already assigned to %q, not selected interface %q", site.Name, ip, gotIface, wantIface)
				}
				prefix, ok, err := ifaceByName(wantIface, ip)
				if err != nil {
					return fmt.Errorf("site %q manage_ip: %w", site.Name, err)
				}
				if site.ManagePrefixLen > 0 {
					prefix, ok = site.ManagePrefixLen, true
				}
				if ok {
					// Explicit interface ownership is deterministic, so restore our
					// in-memory tracking after a process restart.
					m.added[ip] = fmt.Sprintf("%s/%d", wantIface, prefix)
				}
			}
			// Legacy auto-detected addresses are never adopted because ownership
			// cannot be distinguished from Netplan/keepalived/operator state.
			continue
		}
		iface, prefix, ok := "", 0, false
		if selectedInterface != "" {
			iface = selectedInterface
			var err error
			prefix, ok, err = ifaceByName(iface, ip)
			if err != nil {
				return fmt.Errorf("site %q manage_ip: %w", site.Name, err)
			}
			if site.ManagePrefixLen > 0 {
				prefix, ok = site.ManagePrefixLen, true
			}
			if !ok {
				return fmt.Errorf("site %q manage_ip: interface %q has no address-family prefix; set manage_prefix_len explicitly", site.Name, iface)
			}
		} else {
			iface, prefix, ok = ifaceForIP(ip)
			if !ok {
				return fmt.Errorf("site %q manage_ip: no interface subnet matches %s; select manage_interface and manage_prefix_len", site.Name, ip)
			}
		}
		cidr := fmt.Sprintf("%s/%d", ip, prefix)
		if err := runIP("add", cidr, iface); err != nil {
			m.log.Error("manage_ip: assign failed", "ip", cidr, "iface", iface, "err", err)
			return fmt.Errorf("site %q manage_ip: assign %s to %s: %w", site.Name, cidr, iface, err)
		}
		m.added[ip] = fmt.Sprintf("%s/%d", iface, prefix)
		m.log.Info("manage_ip: assigned address to interface", "ip", cidr, "iface", iface)
	}

	// remove IPs we previously added that are no longer wanted
	for ip, meta := range m.added {
		if _, stillWanted := want[ip]; stillWanted {
			continue
		}
		parts := strings.SplitN(meta, "/", 2)
		if len(parts) != 2 {
			delete(m.added, ip)
			continue
		}
		iface, prefix := parts[0], parts[1]
		cidr := ip + "/" + prefix
		if err := runIP("del", cidr, iface); err != nil {
			m.log.Warn("manage_ip: remove failed", "ip", cidr, "iface", iface, "err", err)
		} else {
			m.log.Info("manage_ip: removed address from interface", "ip", cidr, "iface", iface)
		}
		delete(m.added, ip)
	}
	return nil
}

// releaseAll removes every address we added (called on shutdown).
func (m *ipManager) releaseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ip, meta := range m.added {
		parts := strings.SplitN(meta, "/", 2)
		if len(parts) != 2 {
			continue
		}
		_ = runIP("del", ip+"/"+parts[1], parts[0])
	}
	m.added = map[string]string{}
}

// runIP shells out to iproute2: `ip addr <op> <cidr> dev <iface>`.
func runIP(op, cidr, iface string) error {
	cmd := exec.Command("ip", "addr", op, cidr, "dev", iface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		// adding an address that already exists is not an error for our purposes
		if op == "add" && strings.Contains(msg, "File exists") {
			return nil
		}
		return fmt.Errorf("%v: %s", err, msg)
	}
	return nil
}
