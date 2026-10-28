// Copyright (C) 2026 NodeByte LTD

package netguard

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

var ErrBlockedAddress = errors.New("destination is not a public internet address")

var blockedPrefixes = func() []netip.Prefix {
	cidrs := []string{
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"240.0.0.0/4",
		"64:ff9b::/96",
		"64:ff9b:1::/48",
		"2001:db8::/32",
	}

	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}()

func IsPublic(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)

	if !ok {
		return false
	}

	addr = addr.Unmap()

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsMulticast() {
		return false
	}

	for _, p := range blockedPrefixes {
		if p.Contains(addr) {
			return false
		}
	}

	return true
}

func control(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)

	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip == nil || !IsPublic(ip) {
		return ErrBlockedAddress
	}

	return nil
}

func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   control,
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
