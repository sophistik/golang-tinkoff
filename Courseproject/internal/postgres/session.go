package postgres

import (
	"database/sql"

	"../session"
	"github.com/pkg/errors"
)

var _ session.Storage = &SessionStorage{}

type SessionStorage struct {
	statementStorage

	createStmt     *sql.Stmt
	findByIDStmt   *sql.Stmt
	findByTokenStmt   *sql.Stmt
	deleteByIDStmt *sql.Stmt
}

func NewSessionStorage(db *DB) (*SessionStorage, error) {
	s := &SessionStorage{statementStorage: newStatementsStorage(db)}

	stmts := []stmt{
		{Query: createSessionQuery, Dst: &s.createStmt},
		{Query: findSessionByIDQuery, Dst: &s.findByIDStmt},
		{Query: findSessionByTokenQuery, Dst: &s.findByTokenStmt},
		{Query: deleteSessionByIDQuery, Dst: &s.deleteByIDStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, errors.Wrap(err, "can't init statements")
	}

	return s, nil
}

const sessionFields = "session_id, user_id, created_at, valid_until"

func scanSession(scanner sqlScanner, sess *session.Session) error {
	return scanner.Scan(&sess.SessionID, &sess.UserID, &sess.CreatedAt, &sess.ValidUntil)
}

const createSessionQuery = "INSERT INTO session(session_id, user_id) VALUES ($1, $2) RETURNING valid_until"

func (s *SessionStorage) Create(sess *session.Session) error {
	if err := s.createStmt.QueryRow(&sess.SessionID, &sess.UserID).Scan(&sess.ValidUntil); err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const findSessionByIDQuery = "SELECT " + sessionFields + " FROM session WHERE user_id=$1"

func (s *SessionStorage) FindByID(id int64) (*session.Session, error) {
	var sess session.Session

	row := s.findByIDStmt.QueryRow(id)

	if err := scanSession(row, &sess); err != nil {
		return nil, errors.Wrap(err, "can't scan session")
	}

	return &sess, nil
}

const findSessionByTokenQuery = "SELECT " + sessionFields + " FROM session WHERE session_id=$1"

func (s *SessionStorage) FindByToken(token string) (*session.Session, error) {
	var sess session.Session

	row := s.findByTokenStmt.QueryRow(token)

	if err := scanSession(row, &sess); err != nil {
		return nil, errors.Wrap(err, "can't scan session")
	}

	return &sess, nil
}

const deleteSessionByIDQuery = "DELETE FROM session WHERE user_id=$1"

func (s *SessionStorage) DeleteByID(id int64) error {
	_, err := s.deleteByIDStmt.Exec(id)
	if err != nil {
		return errors.Wrap(err, "can't remove session")
	}

	return nil
}
