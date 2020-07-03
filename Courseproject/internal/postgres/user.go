package postgres

import (
	"database/sql"

	"../user"
	"github.com/pkg/errors"
)

var _ user.Storage = &UserStorage{}

type UserStorage struct {
	statementStorage

	createStmt      *sql.Stmt
	findByEmailStmt *sql.Stmt
	findByIDStmt    *sql.Stmt
	updateByIDStmt  *sql.Stmt
}

func NewUserStorage(db *DB) (*UserStorage, error) {
	s := &UserStorage{statementStorage: newStatementsStorage(db)}

	stmts := []stmt{
		{Query: createUserQuery, Dst: &s.createStmt},
		{Query: findUserByEmailQuery, Dst: &s.findByEmailStmt},
		{Query: findUserByIDQuery, Dst: &s.findByIDStmt},
		{Query: updateUserByIDQuery, Dst: &s.updateByIDStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, errors.Wrap(err, "can't init statements")
	}

	return s, nil
}

const userFields = "id, first_name, last_name, birthday, email, password, created_at, updated_at"

func scanUser(scanner sqlScanner, u *user.User) error {
	return scanner.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Birthday, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)
}

const createUserQuery = "INSERT INTO users(first_name, last_name, birthday, email, password) VALUES ($1, $2, $3, $4, $5) RETURNING id"

func (s *UserStorage) Create(u *user.User) error {
	if err := s.createStmt.QueryRow(&u.FirstName, &u.LastName, &u.Birthday, &u.Email, &u.Password).Scan(&u.ID); err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const findUserByEmailQuery = "SELECT " + userFields + " FROM users WHERE email=$1"

func (s *UserStorage) FindByEmail(email string) (*user.User, error) {
	var u user.User

	row := s.findByEmailStmt.QueryRow(email)

	if err := scanUser(row, &u); err != nil {
		return nil, errors.Wrap(err, "can't scan user")
	}

	return &u, nil
}

const findUserByIDQuery = "SELECT " + userFields + " FROM users WHERE id=$1"

func (s *UserStorage) FindByID(id int64) (*user.User, error) {
	var u user.User

	row := s.findByIDStmt.QueryRow(id)

	if err := scanUser(row, &u); err != nil {
		return nil, errors.Wrap(err, "can't scan user")
	}

	return &u, nil
}

const updateUserByIDQuery = "UPDATE users SET (first_name, last_name, birthday, email, password, updated_at) = ($1, $2, $3, $4, $5, now()) WHERE id=$6"

func (s *UserStorage) UpdateByID(u *user.User) error {
	_, err := s.updateByIDStmt.Exec(&u.FirstName, &u.LastName, &u.Birthday, &u.Email, &u.Password, &u.ID)

	if err != nil {
		return errors.Wrap(err, "can't scan user")
	}

	return nil
}
