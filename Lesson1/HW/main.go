package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"strconv"
)

type CandleInfo struct {
	time   string
	opened float64
	higher float64
	lower  float64
	closed float64
}

type profitInfo struct {
	profit     float64
	timeToBuy  string
	timeToSell string
}

func main() {
	candles, err := getCandlesInfo("candles_5m.csv")
	if err != nil {
		log.Fatal(err)
	}

	maxProfit := calculateMaxProfit(candles)

	usersID, usersProfit, err := calculateUsersProfit("user_trades.csv")
	if err != nil {
		log.Fatal(err)
	}

	err = writeData(usersID, maxProfit, usersProfit, "output.csv")
	if err != nil {
		log.Fatal(err)
	}
}

func getCandlesInfo(inputFile string) (map[string][]CandleInfo, error) {
	candles := make(map[string][]CandleInfo)

	csvCandles, err := os.Open(inputFile)
	if err != nil {
		return nil, errors.Wrap(err, "can't open file")
	}

	defer csvCandles.Close()

	candlesInput := csv.NewReader(bufio.NewReader(csvCandles))

	for {
		line, err := candlesInput.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, errors.Wrap(err, "error while reading file")
		}

		ticker := line[0]

		opened, err := strconv.ParseFloat(line[2], 64)
		if err != nil {
			return nil, errors.Wrap(err, "error while parcing opened variable")
		}

		higher, err := strconv.ParseFloat(line[3], 64)
		if err != nil {
			return nil, errors.Wrap(err, "error while parcing higher variable")
		}

		lower, err := strconv.ParseFloat(line[4], 64)
		if err != nil {
			return nil, errors.Wrap(err, "error while parcing lower variaible")
		}

		closed, err := strconv.ParseFloat(line[5], 64)
		if err != nil {
			return nil, errors.Wrap(err, "error while parcing closed variaible")
		}

		_, ok := candles[ticker]
		if !ok {
			candles[ticker] = make([]CandleInfo, 0)
		}

		candles[ticker] = append(candles[ticker], CandleInfo{
			time:   line[1],
			opened: opened,
			higher: higher,
			lower:  lower,
			closed: closed,
		})
	}

	return candles, nil
}

func calculateMaxProfit(candles map[string][]CandleInfo) map[string]profitInfo {
	maxProfit := make(map[string]profitInfo)

	for ticker, candle := range candles {
		for i := 0; i < len(candle); i++ {
			for j := i; j < len(candle); j++ {
				_, ok := maxProfit[ticker]
				if !ok || candle[j].higher-candle[i].lower > maxProfit[ticker].profit {
					maxProfit[ticker] = profitInfo{
						profit:     candle[j].higher - candle[i].lower,
						timeToBuy:  candle[i].time,
						timeToSell: candle[j].time,
					}
				}
			}
		}
	}

	return maxProfit
}

func calculateUsersProfit(inputFile string) ([]int64, map[int64]map[string]float64, error) {
	var usersID []int64

	usersProfit := make(map[int64]map[string]float64)

	csvTrades, err := os.Open(inputFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, "can't open file")
	}

	defer csvTrades.Close()
	tradesReader := csv.NewReader(bufio.NewReader(csvTrades))

	for {
		line, err := tradesReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, nil, errors.Wrap(err, "error while reading file")
		}

		id, err := strconv.ParseInt(line[0], 10, 64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error while parcing id variaible")
		}

		ticker := line[2]

		buyPrice, err := strconv.ParseFloat(line[3], 64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error while parcing ticker variable")
		}

		sellPrice, err := strconv.ParseFloat(line[4], 64)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error while parcing sellPrice variable")
		}

		_, ok := usersProfit[id]
		if !ok {
			usersID = append(usersID, id)
			usersProfit[id] = make(map[string]float64)
		}

		_, ok = usersProfit[id][ticker]
		if !ok {
			usersProfit[id][ticker] = 0
		}

		usersProfit[id][ticker] += sellPrice - buyPrice
	}

	return usersID, usersProfit, nil
}

func writeData(usersID []int64, maxProfit map[string]profitInfo, usersProfit map[int64]map[string]float64, outputFile string) error {
	output, err := os.Create(outputFile)
	if err != nil {
		return errors.Wrap(err, "can't create file")
	}

	defer output.Close()

	writer := csv.NewWriter(output)
	defer writer.Flush()

	for _, id := range usersID {
		for ticker := range maxProfit {
			_, ok := usersProfit[id][ticker]
			if ok {
				diff := maxProfit[ticker].profit - usersProfit[id][ticker]
				data := []string{fmt.Sprintf("%v", id),
					ticker,
					fmt.Sprintf("%.2f", usersProfit[id][ticker]),
					fmt.Sprintf("%.2f", maxProfit[ticker].profit),
					fmt.Sprintf("%.2f", diff),
					maxProfit[ticker].timeToSell,
					maxProfit[ticker].timeToBuy}

				err := writer.Write(data)
				if err != nil {
					return errors.Wrap(err, "error while writing data")
				}
			}
		}
	}

	return nil
}
