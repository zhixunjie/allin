package room

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no O/0/1/I to avoid confusion
const codeLen = 6

// GenerateCode creates a random 6-character room code.
func GenerateCode() (string, error) {
	var b strings.Builder
	b.Grow(codeLen)
	for i := 0; i < codeLen; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		if err != nil {
			return "", err
		}
		b.WriteByte(codeChars[n.Int64()])
	}
	return b.String(), nil
}
