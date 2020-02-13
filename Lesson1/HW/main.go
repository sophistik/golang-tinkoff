package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
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
	tickers := []string{"AAPL", "SBER", "AMZN"}

	candles := getCandlesInfo("candles_5m.csv")
	maxProfit := calculateMaxProfit(tickers, candles)
	usersID, usersProfit := calculateUsersProfit("user_trades.csv")

	writeData(usersID, tickers, maxProfit, usersProfit, "output.csv")
}

func getCandlesInfo(inputFile string) map[string][]CandleInfo {
	candles := map[string][]CandleInfo{
		"AAPL": {},
		"SBER": {},
		"AMZN": {},
	}

	csvCandles, candlesError := os.Open(inputFile)

	checkError("Error while opening \"candles_5m.csv\"", candlesError)

	defer csvCandles.Close()

	candlesReader := csv.NewReader(bufio.NewReader(csvCandles))

	for {
		line, candlesReaderError := candlesReader.Read()
		if candlesReaderError == io.EOF {
			break
		} else if candlesReaderError != nil {
			log.Fatal(candlesReaderError)
		}

		ticker := line[0]
		opened, openedParceError := strconv.ParseFloat(line[2], 64)
		higher, higherParceError := strconv.ParseFloat(line[3], 64)
		lower, lowerParceError := strconv.ParseFloat(line[4], 64)
		closed, closedParceError := strconv.ParseFloat(line[5], 64)

		checkError("Error while parcing", openedParceError)
		checkError("Error while parcing", higherParceError)
		checkError("Error while parcing", lowerParceError)
		checkError("Error while parcing", closedParceError)

		candles[ticker] = append(candles[ticker], CandleInfo{
			time:   line[1],
			opened: opened,
			higher: higher,
			lower:  lower,
			closed: closed,
		})
	}

	return candles
}

func calculateMaxProfit(tickers []string, candles map[string][]CandleInfo) map[string]profitInfo {
	maxProfit := map[string]profitInfo{
		"AAPL": {profit: 0, timeToBuy: "", timeToSell: ""},
		"SBER": {profit: 0, timeToBuy: "", timeToSell: ""},
		"AMZN": {profit: 0, timeToBuy: "", timeToSell: ""},
	}

	for _, ticker := range tickers {
		candle := candles[ticker]
		for i := 0; i < len(candle); i++ {
			for j := i; j < len(candle); j++ {
				if candle[j].higher-candle[i].lower > maxProfit[ticker].profit {
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

func calculateUsersProfit(inputFile string) ([]int64, map[int64]map[string]float64) {
	var usersID []int64

	usersProfit := make(map[int64]map[string]float64)

	csvTrades, tradesError := os.Open(inputFile)
	checkError("Error while opening file", tradesError)

	defer csvTrades.Close()
	tradesReader := csv.NewReader(bufio.NewReader(csvTrades))

	for {
		line, tradesReaderError := tradesReader.Read()
		if tradesReaderError == io.EOF {
			break
		}

		checkError("Error while reading file", tradesReaderError)

		id, idParceError := strconv.ParseInt(line[0], 10, 64)
		ticker := line[2]
		buyPrice, buyPriceParceError := strconv.ParseFloat(line[3], 64)
		sellPrice, sellPriceParceError := strconv.ParseFloat(line[4], 64)

		checkError("Error while parcing", idParceError)
		checkError("Error while parcing", buyPriceParceError)
		checkError("Error while parcing", sellPriceParceError)

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

	return usersID, usersProfit
}

func writeData(usersID []int64, tickers []string, maxProfit map[string]profitInfo, usersProfit map[int64]map[string]float64, outputFile string) {
	output, outputError := os.Create(outputFile)
	checkError("Error while creating file", outputError)

	defer output.Close()
	writer := csv.NewWriter(output)

	// fmt.Println(usersProfit)

	for _, id := range usersID {
		for _, ticker := range tickers {
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

				writerError := writer.Write(data)
				checkError("Error while writing data", writerError)
			}
		}
	}
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}
