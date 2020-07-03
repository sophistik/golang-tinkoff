package database

import (
	"../user"
	"github.com/pkg/errors"
)

var _ user.Storage = &UserStorage{}

type UserStorage struct {
	userDataID    map[int64]*user.User
	sessionDataEmail map[string]*user.User
	size      int64
}

var errExists = errors.New("user already exists")
var errNotFound = errors.New("user not found")

func NewUserStorage() *UserStorage {
	s := &UserStorage{}
	s.userDataID = make(map[int64]*user.User)
	s.sessionDataEmail = make(map[string]*user.User)

	return s
}

func (s *UserStorage) Create(u *user.User) error {
	_, ok := s.sessionDataEmail[u.Email]
	if ok {
		return errExists
	}

	s.size++
	u.ID = s.size
	s.userDataID[u.ID] = u
	s.sessionDataEmail[u.Email] = u

	return nil
}

func (s *UserStorage) FindByEmail(email string) (*user.User, error) {
	u, ok := s.sessionDataEmail[email]
	if !ok {
		return nil, errNotFound
	}

	return u, nil
}

func (s *UserStorage) FindByID(id int64) (*user.User, error) {
	u, ok := s.userDataID[id]
	if !ok {
		return nil, errNotFound
	}

	return u, nil
}

func (s *UserStorage) UpdateByID(u *user.User) error {
	userData, ok := s.userDataID[u.ID]
	if !ok {
		return errNotFound
	}

	delete(s.sessionDataEmail, userData.Email)
	s.userDataID[u.ID] = u
	s.sessionDataEmail[u.Email] = u

	return nil
}
