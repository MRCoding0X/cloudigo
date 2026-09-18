package service

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

const alnumAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomAlnum returns a cryptographically random alphanumeric string of
// length n (used for upload_id, which appears in public URLs).
func RandomAlnum(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alnumAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alnumAlphabet[idx.Int64()]
	}
	return string(b), nil
}

// RandomHex returns a random hex string encoding n random bytes (used for
// secret codes / private ids embedded in download links).
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RandomDigits returns a random numeric code of length n (email verification).
func RandomDigits(n int) (string, error) {
	digits := "0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		b[i] = digits[idx.Int64()]
	}
	return string(b), nil
}
