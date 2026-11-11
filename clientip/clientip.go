// Copyright (C) 2026 NodeByte LTD

package clientip

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	cloudflareHeader  = "CF-Connecting-IP"
	proxyHeader       = "X-Real-IP"
	internalIPHeader  = "X-Omniplex-Client-IP"
	internalKeyHeader = "X-Omniplex-Client-Key"
)

var cloudflare = mustPrefixes(
	"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
	"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
	"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
	"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
	"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
	"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
)

var internalKey = func() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}()

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(cidrs))
	for i, c := range cidrs {
		out[i] = netip.MustParsePrefix(c)
	}
	return out
}

func parse(s string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func host(remoteAddr string) string {
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return h
	}
	return remoteAddr
}

func IsCloudflare(addr netip.Addr) bool {
	for _, p := range cloudflare {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func SetInternal(req *http.Request, ip string) {
	if ip == "" {
		return
	}
	req.Header.Set(internalIPHeader, ip)
	req.Header.Set(internalKeyHeader, internalKey)
}

func Resolve(r *http.Request) string {
	if key := r.Header.Get(internalKeyHeader); key != "" && subtle.ConstantTimeCompare([]byte(key), []byte(internalKey)) == 1 {
		if addr, ok := parse(r.Header.Get(internalIPHeader)); ok {
			return addr.String()
		}
	}

	peer, ok := parse(host(r.RemoteAddr))

	if ok && (peer.IsPrivate() || peer.IsLoopback()) {
		if forwarded, fok := parse(r.Header.Get(proxyHeader)); fok {
			peer = forwarded
		}
	}

	if !ok {
		return host(r.RemoteAddr)
	}

	if IsCloudflare(peer) {
		if visitor, vok := parse(r.Header.Get(cloudflareHeader)); vok {
			return visitor.String()
		}
	}

	return peer.String()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = Resolve(r)
		r.Header.Del(internalIPHeader)
		r.Header.Del(internalKeyHeader)
		next.ServeHTTP(w, r)
	})
}
