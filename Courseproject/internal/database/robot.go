package database

import (
	"time"

	"../robot"
	"github.com/pkg/errors"
)

var _ robot.Storage = &RobotStorage{}

type RobotStorage struct {
	robotDataID map[int64]*robot.Robot
	size        int64
}

var errActivation = errors.New("activation not available now")
var errDeactivation = errors.New("deactivation not available now")

func NewRobotStorage() *RobotStorage {
	s := &RobotStorage{}
	s.robotDataID = make(map[int64]*robot.Robot)

	return s
}

func (s *RobotStorage) Create(r *robot.Robot) error {
	s.size++
	r.RobotID = s.size
	s.robotDataID[s.size] = r

	return nil
}

func (s *RobotStorage) GetAllRobots() ([]*robot.Robot, error) {
	var robotList []*robot.Robot
	for _, r := range s.robotDataID {
		robotList = append(robotList, r)
	}

	return robotList, nil
}

func (s *RobotStorage) GetAllRobotsByOwnerID(id int64) ([]*robot.Robot, error) {
	var robotList []*robot.Robot

	for _, r := range s.robotDataID {
		if r.OwnerUserID == id {
			robotList = append(robotList, r)
		}
	}

	return robotList, nil
}

func (s *RobotStorage) GetAllRobotsByTicker(ticker string) ([]*robot.Robot, error) {
	var robotList []*robot.Robot

	for _, r := range s.robotDataID {
		if r.Ticker == ticker {
			robotList = append(robotList, r)
		}
	}

	return robotList, nil
}

func (s *RobotStorage) GetAllRobotsByOwnerIDAndTicker(id int64, ticker string) ([]*robot.Robot, error) {
	var robotList []*robot.Robot

	for _, r := range s.robotDataID {
		if r.OwnerUserID == id && r.Ticker == ticker {
			robotList = append(robotList, r)
		}
	}

	return robotList, nil
}

func (s *RobotStorage) FindByID(id int64) (*robot.Robot, error) {
	r, ok := s.robotDataID[id]
	if !ok {
		return nil, errNotFound
	}

	return r, nil
}

func (s *RobotStorage) ActivateByID(id int64) error {
	r, ok := s.robotDataID[id]
	if !ok {
		return errNotFound
	}

	if r.IsActive || (r.PlanStart.Before(time.Now()) && r.PlanEnd.After(time.Now())) {
		return errActivation
	}

	r.IsActive = true
	r.ActivatedAt = time.Now()
	s.robotDataID[id] = r

	return nil
}

func (s *RobotStorage) DeactivateByID(id int64) error {
	r, ok := s.robotDataID[id]
	if !ok {
		return errNotFound
	}

	if !r.IsActive || (r.PlanStart.Before(time.Now()) && r.PlanEnd.After(time.Now())) {
		return errDeactivation
	}

	r.IsActive = false
	r.DeactivatedAt = time.Now()
	s.robotDataID[id] = r

	return nil
}

func (s *RobotStorage) DeleteByID(id int64) error {
	_, ok := s.robotDataID[id]
	if !ok {
		return errNotFound
	}

	delete(s.robotDataID, id)

	return nil
}

func (s *RobotStorage) UpdateByID(r *robot.Robot) error {
	_, ok := s.robotDataID[r.RobotID]
	if !ok {
		return errNotFound
	}

	s.robotDataID[r.RobotID] = r

	return nil
}

func (s *RobotStorage) GetRobotsNeedToRun() ([]*robot.Robot, error) {
	var robotList []*robot.Robot

	for _, r := range s.robotDataID {
		if r.IsActive && r.PlanStart.Before(time.Now()) && r.PlanEnd.After(time.Now()) {
			robotList = append(robotList, r)
		}
	}

	return robotList, nil
}

func (s *RobotStorage) ActivateAllRobots() error {
	return nil
}

func (s *RobotStorage) GetWorkingRobotsByTicker(ticker string) ([]*robot.Robot, error) {
	var robotList []*robot.Robot

	return robotList, nil
}
