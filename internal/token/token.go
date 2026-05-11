package token

import (
	"crypto/rand"
	"math/big"
)

const (
	Alphabet      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	DefaultLength = 12
)

func Generate(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}

	out := make([]byte, length)
	max := big.NewInt(int64(len(Alphabet)))

	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = Alphabet[n.Int64()]
	}

	return string(out), nil
}
