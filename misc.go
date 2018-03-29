package baxtep

import (
	"math/rand"
	"time"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserWithNameNotFound  = errors.New("there is no user with this name")
	ErrUserWithEmailNotFound = errors.New("there is no user with this email")
	ErrUserWithIDNotFound    = errors.New("there is no user with this id")
	ErrUserDisabled          = errors.New("user disabled")
	ErrUserBadPassword       = errors.New("bad user password")
	ErrUserNameExist         = errors.New("this user name exist")
	ErrUserEmailExist        = errors.New("this email exist")
	ErrUserSessionNotFound   = errors.New("user session not found")
	ErrUserSessionExpired    = errors.New("user session expired")
)

var passwordRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-!$")

func generateRandomString(lenght int) string {
	b := make([]rune, lenght)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = passwordRunes[rnd.Intn(len(passwordRunes))]
	}
	return string(b)
}

func getPasswordHash(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

