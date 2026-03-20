package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword 返回明文密码的 bcrypt 哈希值。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword 如果明文密码与存储的哈希值匹配则返回 true。
func CheckPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
