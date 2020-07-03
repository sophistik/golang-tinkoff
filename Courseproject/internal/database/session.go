package database

import (
	"time"

	"../session"
)

var _ session.Storage = &SessionStorage{}

type SessionStorage struct {
	sessionDataID    map[int64]*session.Session
	sessionDataToken map[string]*session.Session
}

func NewSessionStorage() *SessionStorage {
	s := &SessionStorage{}
	s.sessionDataID = make(map[int64]*session.Session)
	s.sessionDataToken = make(map[string]*session.Session)

	return s
}

func (s *SessionStorage) Create(sess *session.Session) error {
	sess.CreatedAt = time.Now()
	sess.ValidUntil = sess.CreatedAt.Add(time.Minute * 30) // nolint: gomnd
	s.sessionDataID[sess.UserID] = sess
	s.sessionDataToken[sess.SessionID] = sess

	return nil
}

func (s *SessionStorage) FindByID(id int64) (*session.Session, error) {
	sess, ok := s.sessionDataID[id]
	if !ok {
		return nil, errNotFound
	}

	return sess, nil
}

func (s *SessionStorage) FindByToken(token string) (*session.Session, error) {
	sess, ok := s.sessionDataToken[token]
	if !ok {
		return nil, errNotFound
	}

	return sess, nil
}

func (s *SessionStorage) DeleteByID(id int64) error {
	sess, ok := s.sessionDataID[id]
	if !ok {
		return nil
	}

	delete(s.sessionDataToken, sess.SessionID)
	delete(s.sessionDataID, id)

	return nil
}
