package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
)

type Transaction struct {
	Id    int     `csv:"Id" gorm:"column:id"`
	Date  string  `csv:"Date" gorm:"column:date"`
	Value float64 `csv:"Transaction" gorm:"column:value"`
}

func main() {

	path, _ := filepath.Abs("txns.csv")
	file, err := os.Open(path)

	if err != nil {
		fmt.Printf("Error reading file: %s", err.Error())
	}

	defer file.Close()

	transactions := []*Transaction{}

	if err := gocsv.UnmarshalFile(file, &transactions); err != nil {
		panic(err)
	}

	totalBalance := 0.0
	creditBalance := map[int]float64{}
	debitBalance := map[int]float64{}
	transactionsByMonth := map[int]int{}

	for _, transaction := range transactions {

		month, _ := strconv.Atoi(strings.Split(transaction.Date, "/")[0])

		totalBalance = totalBalance + transaction.Value

		transactionsByMonth[month] += 1

		if transaction.Value > 0 {
			creditBalance[month] += transaction.Value
		}
		if transaction.Value < 0 {
			debitBalance[month] += +transaction.Value
		}

	}

	fmt.Printf("Total Balance is %f\n", totalBalance)

	for month := range transactionsByMonth {
		fmt.Printf("Number of transactions of month %s is %d\n", time.Month(month).String(), transactionsByMonth[month])
		fmt.Printf("Average debit of month %d is %f\n", month, debitBalance[month])
		fmt.Printf("Average credit of month %d is %f\n", month, creditBalance[month])
	}

}
