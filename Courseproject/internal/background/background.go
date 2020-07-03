package background

import (
	"context"
	"sync"
	"time"

	"../robot"
	streamer "../streamer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	Sold = iota
	Bought
)

type Background struct {
	logger        *zap.SugaredLogger
	robotStorage  robot.Storage
	robotsChan    chan robot.Robot
	runningRobots RunningRobots
}

type RunningRobots struct {
	robots map[int64]int
	mutex  *sync.Mutex
}

func (b Background) Updater(r *robot.Robot, resp streamer.TradingService_PriceClient, conn *grpc.ClientConn) {
	go func() {
		for {
			b.logger.Infof("updater")

			price, err := resp.Recv()

			if r.PlanEnd.Before(time.Now()) {
				b.runningRobots.mutex.Lock()
				delete(b.runningRobots.robots, r.RobotID)
				b.runningRobots.mutex.Unlock()
				b.logger.Infof("end")
				conn.Close()

				return
			}

			if err != nil {
				b.runningRobots.mutex.Lock()
				delete(b.runningRobots.robots, r.RobotID)
				b.runningRobots.mutex.Unlock()
				b.logger.Errorf("error while getting price: %v", err)

				break
			}

			robotData, err := b.robotStorage.FindByID(r.RobotID)
			if err != nil {
				b.logger.Errorf("could not find user by ID: %v", err)

				continue
			}

			updated := false

			// fmt.Printf("price: %v\n", price)
			// fmt.Printf("robotData: %v\n", robotData)

			switch status := b.runningRobots.robots[r.RobotID]; status {
			case Sold:
				if robotData.BuyPrice >= price.BuyPrice {
					robotData.FactYield -= price.BuyPrice
					robotData.DealsCount++

					b.logger.Infof("bought")
					b.runningRobots.mutex.Lock()
					b.runningRobots.robots[r.RobotID] = Bought
					b.runningRobots.mutex.Unlock()

					updated = true
				}
			case Bought:
				if robotData.SellPrice <= price.SellPrice {
					robotData.FactYield += price.SellPrice

					b.logger.Infof("sold")
					b.runningRobots.mutex.Lock()
					b.runningRobots.robots[r.RobotID] = Sold
					b.runningRobots.mutex.Unlock()

					updated = true
				}
			}

			if updated {
				b.logger.Infof("updating")
				b.robotsChan <- *robotData

				if err := b.robotStorage.UpdateByID(robotData); err != nil {
					b.logger.Errorf("can't update robot: %+v", err)
				}
			}
		}
	}()
}

func (b Background) Listener(r *robot.Robot) {
	conn, err := grpc.Dial("localhost:5000", grpc.WithInsecure())
	if err != nil {
		b.logger.Errorf("can't connect to server: %v", err)

		return
	}
	// defer conn.Close()

	client := streamer.NewTradingServiceClient(conn)
	req := streamer.PriceRequest{Ticker: r.Ticker}
	resp, err := client.Price(context.Background(), &req)

	if err != nil {
		b.logger.Errorf("can't get price: %v", err)

		return
	}

	b.runningRobots.mutex.Lock()
	b.runningRobots.robots[r.RobotID] = Sold
	b.runningRobots.mutex.Unlock()
	b.Updater(r, resp, conn)
}

func (b Background) RunActivateRobots() {
	go func() {
		for {
			robots, err := b.robotStorage.GetRobotsNeedToRun()
			if err != nil {
				b.logger.Errorf("error during getting robots list: %+v", err)
				break
			}

			if len(robots) != 0 {
				b.logger.Infof("%d robots are running", len(robots))
			}

			for _, v := range robots {
				if _, ok := b.runningRobots.robots[v.RobotID]; !ok {
					b.Listener(v)
				}
			}

			time.Sleep(3 * time.Second) // nolint: gomnd
		}
	}()
}
func NewBackground(logger *zap.SugaredLogger, robotStorage robot.Storage, robotsChan chan robot.Robot) {
	result := Background{logger: logger, robotStorage: robotStorage, robotsChan: robotsChan}
	result.runningRobots.robots = make(map[int64]int)
	result.runningRobots.mutex = new(sync.Mutex)
	result.RunActivateRobots()
}
