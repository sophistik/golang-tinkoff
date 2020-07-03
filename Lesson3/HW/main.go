package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const timeFormat = "2006-01-02 15:04:05.000000"

type InputInfo struct {
	ticker string
	price  float64
	count  int64
	time   time.Time
}

type CandleInfo struct {
	ticker string
	time   time.Time
	opened float64
	higher float64
	lower  float64
	closed float64
}

// nolint: gomnd
func main() {
	var inputFile string

	flag.StringVar(&inputFile, "file", "", "input file")
	flag.Parse()

	if inputFile == "" {
		fmt.Printf("input file required --file=\n")
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(5)

	ch := readData(inputFile, wg)

	out5 := makeCandles(5, ch[0], wg)
	out30 := makeCandles(30, ch[1], wg)
	out240 := makeCandles(240, ch[2], wg)

	candlesLens := []int{5, 30, 240}
	out := []chan CandleInfo{out5, out30, out240}

	result := writeData(out, candlesLens, wg)
	timer := time.NewTimer(5 * time.Second)

	select {
	case <-timer.C:
		fmt.Println("time out")
		return
	case <-result:
		if !timer.Stop() {
			<-timer.C
		}
	}

	wg.Wait()
}

func parceData(line []string) (InputInfo, error) {
	ticker := line[0]

	price, err := strconv.ParseFloat(line[1], 64)
	if err != nil {
		return InputInfo{}, errors.Wrap(err, "error while parcing price variable")
	}

	count, err := strconv.ParseInt(line[2], 10, 64)
	if err != nil {
		return InputInfo{}, errors.Wrap(err, "error while parcing count variable")
	}

	t, err := time.Parse(timeFormat, line[3])
	if err != nil {
		return InputInfo{}, errors.Wrap(err, "error while parcing time variable")
	}

	return InputInfo{
		ticker: ticker,
		price:  price,
		count:  count,
		time:   t,
	}, nil
}

func readData(inputFile string, waiter *sync.WaitGroup) []chan InputInfo {
	ch := make([]chan InputInfo, 3)
	for i := range ch {
		ch[i] = make(chan InputInfo)
	}

	go func(waiter *sync.WaitGroup) {
		defer waiter.Done()

		csvTrades, err := os.Open(inputFile)
		if err != nil {
			log.Println("can't open file", err)
		}

		tradesInput := csv.NewReader(csvTrades)

		for line, err := tradesInput.Read(); err != io.EOF; line, err = tradesInput.Read() {
			if err != nil {
				log.Println("error while reading file", err)
			}

			in, err := parceData(line)
			if err != nil {
				log.Println("cannot parse line", line, err)
			}

			for i := range ch {
				ch[i] <- in
			}
		}

		for i := range ch {
			close(ch[i])
		}

		csvTrades.Close()
	}(waiter)

	return ch
}

// nolint: gomnd
func makeCandles(candleLen int, in <-chan InputInfo, waiter *sync.WaitGroup) chan CandleInfo {
	candles := make(map[string]CandleInfo)
	out := make(chan CandleInfo)

	var timer, beginTime, endTime time.Time

	go func(waiter *sync.WaitGroup) {
		defer waiter.Done()

		for {
			trade, ok := <-in
			timeDuration, _ := time.ParseDuration(fmt.Sprint(candleLen) + "m")

			if beginTime.IsZero() {
				beginTime = time.Date(trade.time.Year(), trade.time.Month(), trade.time.Day(), 7, 0, 0, 0, time.UTC)
				endTime = beginTime.Add(time.Hour * time.Duration(17))
				timer = beginTime
			}

			if !ok {
				writeToChan(out, candles)

				close(out)

				break
			}

			if trade.time.After(endTime) {
				writeToChan(out, candles)

				beginTime = beginTime.AddDate(0, 0, 1)
				endTime = endTime.AddDate(0, 0, 1)
				timer = beginTime
				candles = make(map[string]CandleInfo)
			}

			if trade.time.Before(beginTime) {
				continue
			}

			candle, ok := candles[trade.ticker]
			if !ok && trade.time.Sub(timer) <= timeDuration {
				candle = CandleInfo{trade.ticker, timer, trade.price, trade.price, trade.price, trade.price}
			}

			if trade.time.Sub(candle.time) > timeDuration {
				writeToChan(out, candles)

				candles = make(map[string]CandleInfo)
				timer = timer.Add(time.Minute * time.Duration(candleLen))
				candles[trade.ticker] = CandleInfo{trade.ticker, timer, trade.price, trade.price, trade.price, trade.price}

				continue
			}

			candles[trade.ticker] = CandleInfo{candle.ticker, candle.time, candle.opened, math.Max(candle.higher, trade.price), math.Min(candle.lower, trade.price), trade.price}
		}
	}(waiter)

	return out
}

func writeToChan(out chan CandleInfo, candles map[string]CandleInfo) {
	for _, v := range candles {
		out <- v
	}
}

func writeData(in []chan CandleInfo, candlesLens []int, waiter *sync.WaitGroup) chan struct{} {
	result := make(chan struct{})

	go func(waiter *sync.WaitGroup) {
		defer waiter.Done()

		output := make([]*os.File, cap(candlesLens))
		writer := make([]*csv.Writer, cap(candlesLens))
		ok := make([]bool, cap(candlesLens))

		var err error

		for i := range candlesLens {
			output[i], err = os.Create(fmt.Sprintf("./candles_%dmin.csv", candlesLens[i]))
			if err != nil {
				log.Println("can't create file", err)
				result <- struct{}{}

				return
			}

			writer[i] = csv.NewWriter(output[i])
			defer output[i].Close()
			defer writer[i].Flush()
		}

		var candle CandleInfo

		for {
			select {
			case candle, ok[0] = <-in[0]:
				write(candle, writer[0], ok[0])

			case candle, ok[1] = <-in[1]:
				write(candle, writer[1], ok[1])

			case candle, ok[2] = <-in[2]:
				write(candle, writer[2], ok[2])
			}

			var status bool

			for i := range ok {
				status = status || ok[i]
			}

			if !status {
				result <- struct{}{}
				return
			}
		}
	}(waiter)

	return result
}

func write(candle CandleInfo, writer *csv.Writer, ok bool) {
	if !ok {
		return
	}

	data := []string{
		candle.ticker,
		candle.time.Format(time.RFC3339Nano),
		fmt.Sprintf("%v", candle.opened),
		fmt.Sprintf("%v", candle.higher),
		fmt.Sprintf("%v", candle.lower),
		fmt.Sprintf("%v", candle.closed)}

	err := writer.Write(data)

	if err != nil {
		log.Println("error while writing data", err)
	}
}
