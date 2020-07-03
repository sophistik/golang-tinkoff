package postgres

import (
	"log"
	"testing"
	"time"

	"../robot"

	"go.uber.org/zap"
)

//[s@Ms postgres]$ go test -v -bench .
// goos: linux
// goarch: amd64
// BenchmarkCreateRobot-4    	1000000000	         0.0294 ns/op
// BenchmarkGetAllRobots-4   	1000000000	         0.0155 ns/op
// PASS
// ok  	_/home/s/MAI/tinkoff/tfs-go-courseproject/internal/postgres	0.363s

// Исходя из данного бенчмарка, можно сделать вывод, что получение списка
// роботов происходит даже быстрее, чем создание. Можно сделать вывод, что
// получение списка работает оптимально

func BenchmarkCreateRobot(b *testing.B) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Can't create zap logger: ", err)
	}

	cfg := Config{"postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable&fallback_application_name=courseproject", 2, time.Minute * 5}

	db, _ := New(logger, cfg)
	robotStorage, _ := NewRobotStorage(db)
	r := robot.Robot{Ticker: "SBER"}

	for i := 0; i < 10; i++ {
		_ = robotStorage.Create(&r)
	}
}

func BenchmarkGetAllRobots(b *testing.B) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Can't create zap logger: ", err)
	}

	cfg := Config{"postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable&fallback_application_name=courseproject", 2, time.Minute * 5}

	db, _ := New(logger, cfg)
	robotStorage, _ := NewRobotStorage(db)
	_, _ = robotStorage.GetAllRobots()
}
