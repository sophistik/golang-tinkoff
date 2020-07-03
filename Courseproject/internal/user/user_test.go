package user

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testCase struct {
	U  *User
	OK bool
}

func Test_CheckCorrectData_True(t *testing.T) {
	r := require.New(t)
	u := &User{FirstName: "Correct", LastName: "Data", Email: "correct@mail.ru", Password: "correct_pass"}
	tc := testCase{u, true}

	r.Equal(tc.OK, tc.U.CheckCorrectData())
}

func Test_CheckPasswordHash(t *testing.T) {
	r := require.New(t)
	tc := "password"
	hashedPass, err := HashPassword(tc)

	r.NoError(err)


	r.Equal(CheckPasswordHash(tc, hashedPass), true)
}
