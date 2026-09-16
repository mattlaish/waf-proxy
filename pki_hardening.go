package main

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

// PKI Slice 3 hardening helpers. These are deliberately independent from
// certificate/key management. They only protect CRL retrieval boundaries.

type crlFetchPolicy struct {
	AllowHTTP bool
}

func validateCRLFetchURL(raw string, policy crlFetchPolicy) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid CRL URL")
	}
	if u.Scheme != "https" && !(policy.AllowHTTP && u.Scheme == "http") {
		return errors.New("CRL URL scheme is not allowed")
	}
	if u.Host == "" || u.User != nil {
		return errors.New("CRL URL host is invalid")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("CRL URL hostname is empty")
	}
	if strings.EqualFold(host, "localhost") {
		return errors.New("CRL URL localhost is blocked")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return errors.New("CRL URL private or local address is blocked")
		}
	}
	return nil
}

type crlRefreshState string

const (
	crlRefreshPending crlRefreshState = "PENDING"
	crlRefreshRunning crlRefreshState = "RUNNING"
	crlRefreshSuccess crlRefreshState = "SUCCESS"
	crlRefreshFailed  crlRefreshState = "FAILED"
)

type crlRefreshRecord struct {
	URL        string
	State      crlRefreshState
	Attempts   int
	LastError  string
	UpdatedAt  time.Time
	ActiveHash string
}

// lastKnownGoodCRL retains the previous accepted CRL material when a refresh
// candidate fails validation.
type lastKnownGoodCRL struct {
	LoadedAt time.Time
	Data     []byte
}

func retainLastKnownGood(previous *lastKnownGoodCRL, candidate []byte, now time.Time, valid bool) *lastKnownGoodCRL {
	if valid {
		return &lastKnownGoodCRL{LoadedAt: now, Data: append([]byte(nil), candidate...)}
	}
	if previous == nil {
		return nil
	}
	return previous
}
