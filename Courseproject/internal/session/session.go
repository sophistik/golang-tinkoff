package session

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

type Session struct {
	SessionID  string
	UserID     int64
	CreatedAt  time.Time
	ValidUntil time.Time
}

type Storage interface {
	Create(sess *Session) error
	FindByID(id int64) (*Session, error)
	FindByToken(token string) (*Session, error)
	DeleteByID(id int64) error
}

const alphabet = "qwertyuiopasdfghjlzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890"

const tokenLen = 20

func GenerateToken() (string, error) {
	result := make([]uint8, tokenLen)

	for i := 0; i < tokenLen; i++ {
		res, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", errors.Wrapf(err, "can't generate token")
		}

		result[i] = alphabet[res.Int64()]
	}

	return string(result), nil
}
