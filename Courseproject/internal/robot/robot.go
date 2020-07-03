package robot

import (
	"time"

	"../../pkg/null"
)

type Robot struct {
	RobotID       int64         `json:"robot_id"`
	OwnerUserID   int64         `json:"owner_user_id"`
	ParentRobotID int64         `json:"parent_robot_id"`
	IsFavorite    bool          `json:"is_favorite"`
	IsActive      bool          `json:"is_active"`
	Ticker        string        `json:"ticker"`
	BuyPrice      float64       `json:"buy_price"`
	SellPrice     float64       `json:"sell_price"`
	PlanStart     time.Time     `json:"plan_start"`
	PlanEnd       time.Time     `json:"plan_end"`
	PlanYield     float64       `json:"plan_yield"`
	FactYield     float64       `json:"fact_yield"`
	DealsCount    int64         `json:"deals_count"`
	ActivatedAt   time.Time     `json:"activated_at"`
	DeactivatedAt time.Time     `json:"deactivated_at"`
	CreatedAt     time.Time     `json:"created_at"`
	DeletedAt     null.NullTime `json:"deleted_at"`
}

type Storage interface {
	Create(r *Robot) error
	GetAllRobots() ([]*Robot, error)
	GetAllRobotsByOwnerID(id int64) ([]*Robot, error)
	GetAllRobotsByTicker(ticker string) ([]*Robot, error)
	GetAllRobotsByOwnerIDAndTicker(id int64, ticker string) ([]*Robot, error)
	FindByID(id int64) (*Robot, error)
	ActivateByID(id int64) error
	DeactivateByID(id int64) error
	DeleteByID(id int64) error
	UpdateByID(r *Robot) error
	GetRobotsNeedToRun() ([]*Robot, error)
	ActivateAllRobots() error
	GetWorkingRobotsByTicker(ticker string) ([]*Robot, error)
}
