package netguard

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsPublic(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.0.0.2", "0.0.0.0", "10.1.2.3", "172.16.0.1",
		"192.168.1.1", "169.254.169.254", "100.64.0.1", "::1", "::",
		"fc00::1", "fe80::1", "::ffff:127.0.0.1", "::ffff:10.0.0.1",
		"64:ff9b::a00:1", "224.0.0.1", "255.255.255.255",
	}

	for _, s := range blocked {
		if IsPublic(net.ParseIP(s)) {
			t.Errorf("IsPublic(%s) = true, want false", s)
		}
	}

	for _, s := range []string{"1.1.1.1", "8.8.8.8", "162.159.128.233", "2606:4700:4700::1111"} {
		if !IsPublic(net.ParseIP(s)) {
			t.Errorf("IsPublic(%s) = false, want true", s)
		}
	}
}

func TestClientRefusesLocalServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("internal"))
	}))
	defer srv.Close()

	_, err := NewClient(5 * time.Second).Get(srv.URL)

	if !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("err = %v, want ErrBlockedAddress", err)
	}
}
