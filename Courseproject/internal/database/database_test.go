package database

import (
	"testing"
	"time"

	"../robot"

	"github.com/stretchr/testify/require"
)

// nolint: gomnd
func Test_AcivateRobot(t *testing.T) {
	r := require.New(t)
	tc := &robot.Robot{PlanStart: time.Now().Add(3 * time.Minute), PlanEnd: time.Now().Add(4 * time.Minute)}
	s := NewRobotStorage()

	err := s.Create(tc)
	r.NoError(err)

	err = s.ActivateByID(1)

	r.NoError(err)
}

// nolint: gomnd
func Test_DeacivateRobot(t *testing.T) {
	r := require.New(t)
	tc := &robot.Robot{IsActive: true, PlanStart: time.Now().Add(3 * time.Minute), PlanEnd: time.Now().Add(4 * time.Minute)}
	s := NewRobotStorage()

	err := s.Create(tc)
	r.NoError(err)

	err = s.DeactivateByID(1)

	r.NoError(err)
}
