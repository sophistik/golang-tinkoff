package user

import (
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Birthday  time.Time `json:"birthday"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ShortUser struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Birthday  time.Time `json:"birthday"`
}

type Storage interface {
	Create(u *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id int64) (*User, error)
	UpdateByID(*User) error
}

func (u *User) CheckCorrectData() bool {
	uncorrect := u.FirstName == "" || u.LastName == "" || u.Email == "" || u.Password == ""

	return !uncorrect
}

func HashPassword(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", errors.Wrapf(err, "can't hash password %s", password)
	}

	return string(passwordHash), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
