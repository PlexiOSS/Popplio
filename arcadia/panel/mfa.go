// Copyright (C) 2026 NodeByte LTD

package panel

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	totpSecretBits = 160
	otpLabel       = "staff@infinitybots.gg"
	otpIssuer      = "Omniplex"
	qrSize         = 200
)

func generateTOTPSecret() (string, error) {
	secret := make([]byte, totpSecretBits/8)

	if _, err := rand.Read(secret); err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString(secret), nil
}

func otpAuthURI(secret string) string {
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", otpIssuer)

	return fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(otpLabel), q.Encode())
}

func verifyTOTP(code, secret string) (bool, error) {
	return totp.ValidateCustom(strings.TrimSpace(code), secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      0,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
}

func renderQRSVG(content string) (string, error) {
	qr, err := qrcode.New(content, qrcode.Medium)

	if err != nil {
		return "", err
	}

	bitmap := qr.Bitmap()
	modules := len(bitmap)

	if modules == 0 {
		return "", fmt.Errorf("qr code produced an empty bitmap")
	}

	var sb strings.Builder

	fmt.Fprintf(&sb,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" shape-rendering="crispEdges">`,
		qrSize, qrSize, modules, modules)

	sb.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	sb.WriteString(`<path fill="#000000" d="`)

	for y, row := range bitmap {
		x := 0

		for x < len(row) {
			if !row[x] {
				x++
				continue
			}

			start := x

			for x < len(row) && row[x] {
				x++
			}

			fmt.Fprintf(&sb, "M%d %dh%dv1H%dz", start, y, x-start, start)
		}
	}

	sb.WriteString(`"/></svg>`)

	return sb.String(), nil
}
