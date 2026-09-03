// Copyright (C) 2026 NodeByte LTD

package sender

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"

	"popplio/state"

	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/crypto"
)

const userAgent = "Popplio/v8.0.0 (https://infinitybots.gg)"

const dnsTimeout = 5 * time.Second

var webhookClient = &http.Client{
	Timeout: 30 * time.Second,
}

type Secret struct {
	Raw string
}

func (s Secret) Sign(data []byte) string {
	h := hmac.New(sha512.New, []byte(s.Raw))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (st *webhookSendState) resolveTarget(webhookURL string) error {
	if len(st.ResolvedIps) == 0 {
		parsed, err := url.ParseRequestURI(webhookURL)

		if err != nil {
			st.cancelSend("INVALID_REQUEST_URL")
			return err
		}

		timeoutCtx, cancel := context.WithTimeout(state.Context, dnsTimeout)
		defer cancel()

		ips, err := net.DefaultResolver.LookupHost(timeoutCtx, parsed.Hostname())

		if err != nil {
			st.cancelSend("CNAME_LOOKUP_FAILURE")
			return err
		}

		st.ResolvedIps = ips
	}

	state.Logger.Info("Resolved webhook IP", st.logFields(zap.Strings("resolvedIp", st.ResolvedIps))...)

	if slices.Contains(st.ResolvedIps, "127.0.0.1") {
		st.cancelSend("LOCALHOST_URL")
		return errors.New("localhost url")
	}

	return nil
}

func (st *webhookSendState) buildRequest(webhook *webhookData, data []byte) (*http.Request, error) {
	secret := webhook.Secret

	if st.BadIntent {
		secret = crypto.RandString(128)
	}

	switch {
	case webhook.HmacAuth:
		return buildHmacAuthRequest(webhook.Url, secret, data)
	case webhook.SimpleAuth:
		return buildSimpleAuthRequest(webhook.Url, secret, data)
	default:
		return buildSplashtailRequest(webhook.Url, secret, data)
	}
}

func buildHmacAuthRequest(url, secret string, data []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(state.Context, "POST", url, bytes.NewReader(data))

	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	sig := hex.EncodeToString(h.Sum(nil))

	req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	req.Header.Set("X-Webhook-Protocol", "hmac-sha256")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	return req, nil
}

func buildSimpleAuthRequest(url, secret string, data []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(state.Context, "POST", url, bytes.NewReader(data))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", secret)
	req.Header.Set("X-Webhook-Protocol", "simple-auth")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", userAgent)

	return req, nil
}

func buildSplashtailRequest(url, secret string, data []byte) (*http.Request, error) {
	postData, nonce, token, err := sealPayload(secret, data)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(state.Context, "POST", url, bytes.NewReader(postData))

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Webhook-Signature", token)
	req.Header.Set("X-Webhook-Protocol", "splashtail")
	req.Header.Set("X-Webhook-Nonce", nonce)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", userAgent)

	return req, nil
}

func sealPayload(secret string, data []byte) (postData []byte, nonce, token string, err error) {
	nonce = crypto.RandString(16)

	keyHash := sha256.New()
	keyHash.Write([]byte(secret + nonce))

	block, err := aes.NewCipher(keyHash.Sum(nil))

	if err != nil {
		return nil, "", "", err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, "", "", err
	}

	aesNonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, aesNonce); err != nil {
		return nil, "", "", err
	}

	postData = []byte(hex.EncodeToString(gcm.Seal(aesNonce, aesNonce, data, nil)))

	inner := Secret{Raw: secret}.Sign(postData)
	token = Secret{Raw: nonce}.Sign([]byte(inner))

	return postData, nonce, token, nil
}
