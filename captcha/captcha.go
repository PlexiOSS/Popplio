package captcha

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"popplio/db"
	"popplio/state"
)

const Difficulty = 20
const Lifetime = 2 * time.Minute
const saltBytes = 16

type Challenge struct {
	Salt       string `json:"salt"`
	Difficulty int    `json:"difficulty"`
	Expires    int64  `json:"expires"`
	Signature  string `json:"signature"`
}

type Solution struct {
	Salt       string `json:"salt"`
	Difficulty int    `json:"difficulty"`
	Expires    int64  `json:"expires"`
	Signature  string `json:"signature"`
	Nonce      string `json:"nonce"`
}

func secret() []byte {
	return []byte(state.Config.Captcha.HMACSecret)
}

func sign(salt string, difficulty int, expires int64) string {
	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(payloadToSign(salt, difficulty, expires)))
	return hex.EncodeToString(mac.Sum(nil))
}

func payloadToSign(salt string, difficulty int, expires int64) string {
	return salt + "|" + strconv.Itoa(difficulty) + "|" + strconv.FormatInt(expires, 10)
}

func New() (*Challenge, error) {
	saltRaw := make([]byte, saltBytes)
	if _, err := rand.Read(saltRaw); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltRaw)
	expires := time.Now().Add(Lifetime).UnixMilli()

	return &Challenge{
		Salt:       salt,
		Difficulty: Difficulty,
		Expires:    expires,
		Signature:  sign(salt, Difficulty, expires),
	}, nil
}

func leadingZeroBits(h [32]byte) int {
	count := 0
	for _, b := range h {
		if b == 0 {
			count += 8
			continue
		}
		count += bitsLeadingZero(b)
		break
	}
	return count
}

func bitsLeadingZero(b byte) int {
	n := 0
	for i := 7; i >= 0; i-- {
		if (b>>uint(i))&1 != 0 {
			break
		}
		n++
	}
	return n
}

func Verify(ctx context.Context, s Solution) error {
	if s.Salt == "" || s.Signature == "" || s.Nonce == "" {
		return errors.New("captcha solution is missing required fields")
	}

	expectedSig := sign(s.Salt, s.Difficulty, s.Expires)
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(s.Signature)) != 1 {
		return errors.New("captcha challenge signature is invalid")
	}

	if time.Now().UnixMilli() > s.Expires {
		return errors.New("captcha challenge has expired, please try again")
	}

	if len(s.Nonce) > 32 {
		return errors.New("captcha solution is invalid")
	}
	if _, ok := new(big.Int).SetString(s.Nonce, 10); !ok {
		return errors.New("captcha solution is invalid")
	}

	sum := sha256.Sum256([]byte(s.Salt + ":" + s.Nonce))
	if leadingZeroBits(sum) < s.Difficulty {
		return errors.New("captcha proof of work is insufficient")
	}

	ttl := time.Until(time.UnixMilli(s.Expires))
	if ttl <= 0 {
		ttl = time.Second
	}

	key := "captcha:used:" + s.Signature
	ok, err := state.Redis.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to check captcha replay state: %w", err)
	}
	if !ok {
		return errors.New("captcha solution has already been used")
	}

	return nil
}

func RequiresCaptcha(ctx context.Context, targetType, targetId string) (bool, error) {
	q := db.New(state.Pool)

	var optOut bool
	var err error

	switch targetType {
	case "bot":
		optOut, err = q.GetBotCaptchaOptOut(ctx, targetId)
	case "server":
		optOut, err = q.GetServerCaptchaOptOut(ctx, targetId)
	default:
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check captcha_opt_out: %w", err)
	}

	return !optOut, nil
}

var _ = strings.TrimSpace
