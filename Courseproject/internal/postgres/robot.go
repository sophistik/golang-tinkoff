package postgres

import (
	"database/sql"

	"../robot"
	"github.com/pkg/errors"
)

var _ robot.Storage = &RobotStorage{}

type RobotStorage struct {
	statementStorage

	createStmt                         *sql.Stmt
	getAllRobotsStmt                   *sql.Stmt
	getAllRobotsByOwnerIDAndTickerStmt *sql.Stmt
	getAllRobotsByOwnerIDStmt          *sql.Stmt
	getAllRobotsByTickerStmt           *sql.Stmt
	findByIDStmt                       *sql.Stmt
	deleteByIDStmt                     *sql.Stmt
	updateByIDStmt                     *sql.Stmt
	activateByIDStmt                   *sql.Stmt
	deactivateByIDStmt                 *sql.Stmt
	getRobotsNeedToActivateStmt        *sql.Stmt
	activateAllRobotsStmt              *sql.Stmt
	GetWorkingRobotsByTickerStmt       *sql.Stmt
}

func NewRobotStorage(db *DB) (*RobotStorage, error) {
	s := &RobotStorage{statementStorage: newStatementsStorage(db)}

	stmts := []stmt{
		{Query: createRobotQuery, Dst: &s.createStmt},
		{Query: findAllRobotsQuery, Dst: &s.getAllRobotsStmt},
		{Query: findAllRobotsByOwnerIDAndTickerQuery, Dst: &s.getAllRobotsByOwnerIDAndTickerStmt},
		{Query: findAllRobotsByOwnerIDQuery, Dst: &s.getAllRobotsByOwnerIDStmt},
		{Query: findAllRobotsByTickerQuery, Dst: &s.getAllRobotsByTickerStmt},
		{Query: findRobotByIDQuery, Dst: &s.findByIDStmt},
		{Query: deleteRobotByIDQuery, Dst: &s.deleteByIDStmt},
		{Query: updateRobotByIDQuery, Dst: &s.updateByIDStmt},
		{Query: activateRobotByIDQuery, Dst: &s.activateByIDStmt},
		{Query: deactivateRobotByIDQuery, Dst: &s.deactivateByIDStmt},
		{Query: getRobotsNeedToActivateQuery, Dst: &s.getRobotsNeedToActivateStmt},
		{Query: activateAllRobotsQuery, Dst: &s.activateAllRobotsStmt},
		{Query: GetWorkingRobotsByTickerQuery, Dst: &s.GetWorkingRobotsByTickerStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, errors.Wrap(err, "can't init statements")
	}

	return s, nil
}

const robotFields = "robot_id, owner_user_id, parent_robot_id, is_favorite, is_active, ticker, buy_price, " +
	"sell_price, plan_start, plan_end, fact_yield, deals_count, activated_at, deactivated_at, created_at"

func scanRobot(scanner sqlScanner, r *robot.Robot) error {
	return scanner.Scan(&r.RobotID, &r.OwnerUserID, &r.ParentRobotID, &r.IsFavorite, &r.IsActive, &r.Ticker, &r.BuyPrice, &r.SellPrice, &r.PlanStart, &r.PlanEnd, &r.PlanYield, &r.FactYield, &r.DealsCount, &r.ActivatedAt, &r.DeactivatedAt, &r.CreatedAt, &r.DeletedAt)
}

const createRobotQuery = "INSERT INTO robots(owner_user_id, parent_robot_id, is_favorite, ticker) VALUES ($1, $2, $3, $4) RETURNING robot_id"

func (s *RobotStorage) Create(r *robot.Robot) error {
	if err := s.createStmt.QueryRow(&r.OwnerUserID, &r.ParentRobotID, &r.IsFavorite, &r.Ticker).Scan(&r.RobotID); err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const findAllRobotsQuery = "SELECT * FROM robots WHERE deleted_at IS NULL"

func (s *RobotStorage) GetAllRobots() ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.getAllRobotsStmt.Query()

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		var r robot.Robot

		if err := scanRobot(rows, &r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, &r)
	}

	return robots, nil
}

const findAllRobotsByOwnerIDAndTickerQuery = "SELECT * FROM robots WHERE deleted_at IS NULL AND owner_user_id=$1 AND ticker=$2"

func (s *RobotStorage) GetAllRobotsByOwnerIDAndTicker(id int64, ticker string) ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.getAllRobotsByOwnerIDAndTickerStmt.Query(id, ticker)

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		var r robot.Robot

		if err := scanRobot(rows, &r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, &r)
	}

	return robots, nil
}

const findAllRobotsByOwnerIDQuery = "SELECT * FROM robots WHERE deleted_at IS NULL AND owner_user_id=$1"

func (s *RobotStorage) GetAllRobotsByOwnerID(id int64) ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.getAllRobotsByOwnerIDStmt.Query(id)

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		var r robot.Robot

		if err := scanRobot(rows, &r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, &r)
	}

	return robots, nil
}

const findAllRobotsByTickerQuery = "SELECT * FROM robots WHERE deleted_at IS NULL AND ticker=$1"

func (s *RobotStorage) GetAllRobotsByTicker(ticker string) ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.getAllRobotsByTickerStmt.Query(ticker)

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		var r robot.Robot

		if err := scanRobot(rows, &r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, &r)
	}

	return robots, nil
}

const findRobotByIDQuery = "SELECT * FROM robots WHERE robot_id=$1"

func (s *RobotStorage) FindByID(id int64) (*robot.Robot, error) {
	var r robot.Robot

	row := s.findByIDStmt.QueryRow(id)

	if err := scanRobot(row, &r); err != nil {
		return nil, errors.Wrap(err, "can't scan user")
	}

	return &r, nil
}

const deleteRobotByIDQuery = "UPDATE robots SET deleted_at = now() WHERE robot_id=$1"

func (s *RobotStorage) DeleteByID(id int64) error {
	_, err := s.deleteByIDStmt.Exec(id)
	if err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const updateRobotByIDQuery = "UPDATE robots SET (owner_user_id, parent_robot_id, is_favorite, is_active, ticker, buy_price, " +
	"sell_price, plan_start, plan_end, plan_yield, fact_yield, deals_count, activated_at, deactivated_at, created_at, deleted_at) = ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) WHERE robot_id=$17"

func (s *RobotStorage) UpdateByID(r *robot.Robot) error {
	_, err := s.updateByIDStmt.Exec(&r.OwnerUserID, &r.ParentRobotID, &r.IsFavorite, &r.IsActive, &r.Ticker, &r.BuyPrice, &r.SellPrice, &r.PlanStart, &r.PlanEnd, &r.PlanYield, &r.FactYield, &r.DealsCount, &r.ActivatedAt, &r.DeactivatedAt, &r.CreatedAt, &r.DeletedAt, &r.RobotID)

	if err != nil {
		return errors.Wrap(err, "can't scan user")
	}

	return nil
}

const activateRobotByIDQuery = "UPDATE robots SET (activated_at, is_active) = (now(), true) WHERE robot_id=$1"

func (s *RobotStorage) ActivateByID(id int64) error {
	_, err := s.activateByIDStmt.Exec(id)

	if err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const deactivateRobotByIDQuery = "UPDATE robots SET (deactivated_at, is_active) = (now(), false) WHERE robot_id=$1"

func (s *RobotStorage) DeactivateByID(id int64) error {
	_, err := s.deactivateByIDStmt.Exec(id)

	if err != nil {
		return errors.Wrap(err, "can't scan user")
	}

	return nil
}

const getRobotsNeedToActivateQuery = "SELECT * FROM robots WHERE deleted_at IS NULL AND is_active=true AND plan_start < now() AND plan_end > now()"

func (s *RobotStorage) GetRobotsNeedToRun() ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.getRobotsNeedToActivateStmt.Query()

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		r := new(robot.Robot)

		if err := scanRobot(rows, r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, r)
	}

	return robots, nil
}

const activateAllRobotsQuery = "UPDATE robots SET is_active = true WHERE deleted_at IS NULL AND is_active=false AND activated_at < now() AND deactivated_at > now()"

func (s *RobotStorage) ActivateAllRobots() error {
	_, err := s.activateByIDStmt.Exec()
	if err != nil {
		return errors.Wrap(err, "can't exec query")
	}

	return nil
}

const GetWorkingRobotsByTickerQuery = "SELECT * FROM robots WHERE deleted_at IS NULL AND ticker=$1 AND is_active=true AND activated_at < now() AND deactivated_at > now()"

func (s *RobotStorage) GetWorkingRobotsByTicker(ticker string) ([]*robot.Robot, error) {
	robots := make([]*robot.Robot, 0)
	rows, err := s.GetWorkingRobotsByTickerStmt.Query(ticker)

	if err != nil {
		return nil, errors.Wrap(err, "can't exec query")
	}

	for rows.Next() {
		var r *robot.Robot

		if err := scanRobot(rows, r); err != nil {
			return nil, errors.Wrap(err, "can't scan robot")
		}

		robots = append(robots, r)
	}

	return robots, nil
}
