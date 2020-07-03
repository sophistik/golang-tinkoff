package session

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GenerateToken(t *testing.T) {
	r := require.New(t)
	str, err := GenerateToken()
	r.NoError(err)
	r.Equal(len(str), tokenLen)
}
