package room

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除 O/0/1/I 以避免混淆
const codeLen = 6

// GenerateCode 创建一个随机的 6 位房间码。
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
