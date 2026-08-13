/**
* The captcha package is designed to be stateless and self-contained, providing a secure
* and efficient way to prevent automated abuse of voting systems.
*
* It leverages cryptographic techniques to ensure that challenges cannot be tampered
* with or reused, while also being easy to implement and scale.
*
* Key Features:
* - Stateless Challenge Issuance: Challenges are generated without any server-side storage, making it easy to scale and distribute.
* - Proof-of-Work Mechanism: Clients must perform real computational work to solve challenges, deterring automated scripts from abusing the voting system.
* - HMAC Signing: Challenges are signed with a server-only secret, ensuring that they cannot be forged or tampered with.
* - Single-Use Verification: Each solved challenge can only be used once, preventing replay attacks and ensuring the integrity of the voting process.
*
* Usage:
* 1. Generate a new challenge using the New() function.
* 2. Send the challenge to the client for solving.
* 3. Upon receiving a solution from the client, verify it using the Verify() function.
* 4. Check if a captcha is required for a specific target using RequiresCaptcha().
*
* This package is intended for use in environments where preventing automated abuse
* is critical, such as voting systems, and can be easily integrated into existing
* applications.
 */
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

	"popplio/state"
)

const Difficulty = 20
const Lifetime = 2 * time.Minute
const saltBytes = 16

type Challenge struct {
	Salt       string `json:"salt"`
	Difficulty int    `json:"difficulty"`
	Expires    int64  `json:"expires"` // unix ms
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
	return []byte(state.Config.Captcha.HMACSecret.Parse())
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

/**
* Verify checks a client-submitted Solution to ensure it is valid and has not been tampered with or reused.
*
* Parameters:
* - ctx: The context for the verification process, allowing for cancellation and timeouts.
* - s: The Solution object submitted by the client, containing the salt, difficulty, expiration time, signature, and nonce.
*
* Returns:
* - error: An error if the solution is invalid, expired, or has already been used; nil if the solution is valid.
*
* This function performs several checks:
* 1. Validates that all required fields in the Solution are present.
* 2. Verifies that the signature matches the expected value based on the salt, difficulty, and expiration time.
* 3. Checks that the challenge has not expired.
* 4. Ensures that the nonce is a valid number and meets size constraints.
* 5. Confirms that the proof-of-work is sufficient based on the leading zero bits in the hash of the salt and nonce.
* 6. Marks the signature as used in Redis to prevent replay attacks, with a TTL matching the challenge's remaining lifetime.
 */
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

/**
* RequiresCaptcha checks if a captcha is required for voting on a specific target entity.
*
* Parameters:
* - ctx: The context for the database query.
* - targetType: The type of the target entity (e.g., "bot", "server").
* - targetId: The unique identifier of the target entity.
*
* Returns:
* - bool: True if a captcha is required, false otherwise.
* - error: An error if the check fails due to database issues or invalid input.
*
* This function queries the database to determine if the specified target entity
* has opted out of captcha requirements. If the entity has opted out, it returns
* false; otherwise, it returns true, indicating that a captcha is required for voting.
 */
func RequiresCaptcha(ctx context.Context, targetType, targetId string) (bool, error) {
	var table, column string
	switch targetType {
	case "bot":
		table, column = "bots", "bot_id"
	case "server":
		table, column = "servers", "server_id"
	default:
		return false, nil
	}

	var optOut bool
	err := state.Pool.QueryRow(ctx, "SELECT captcha_opt_out FROM "+table+" WHERE "+column+" = $1", targetId).Scan(&optOut)
	if err != nil {
		return false, fmt.Errorf("failed to check captcha_opt_out: %w", err)
	}

	return !optOut, nil
}

var _ = strings.TrimSpace
