package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
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

func (Transaction) TableName() string {
	return "transactions"
}

type TemplateData struct {
	Rows    []MonthData
	Balance float64
}

type MonthData struct {
	Month        string
	Transactions int
	DebitAmount  float64
	CreditAmount float64
}

func main() {

	path, _ := filepath.Abs("txns.csv")
	file, err := os.Open(path)

	if err != nil {
		fmt.Printf("Error reading file: %s", err.Error())
	}

	defer file.Close()

	templateData, err := processCSV(file)

	if err != nil {
		fmt.Printf("Error processing file: %s", err.Error())
	}

	err = sendEmail(templateData)

	if err != nil {
		fmt.Printf("Error processing file: %s", err.Error())
	}
}

func processCSV(csvFile io.ReadCloser) (TemplateData, error) {
	defer csvFile.Close()

	transactions := []*Transaction{}

	if err := gocsv.Unmarshal(csvFile, &transactions); err != nil {
		return TemplateData{}, fmt.Errorf("Error parsing file: %w", err)
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

	monthData := make([]MonthData, 0)

	sortedMonths := make([]int, 0)
	for k, _ := range transactionsByMonth {
		sortedMonths = append(sortedMonths, k)
	}

	sort.Ints(sortedMonths)

	for month := range sortedMonths {
		data := MonthData{
			Month:        time.Month(month).String(),
			Transactions: transactionsByMonth[month],
			DebitAmount:  debitBalance[month],
			CreditAmount: creditBalance[month],
		}
		monthData = append(monthData, data)
	}

	return TemplateData{Rows: monthData, Balance: totalBalance}, nil
}

func sendEmail(data TemplateData) error {

	from := os.Getenv("EMAIL_USER")
	pass := os.Getenv("EMAIL_PASS")
	to := os.Getenv("EMAIL_TO")
	subject := "Account balance"

	renderedBody := new(bytes.Buffer)

	emailTemplate, err := ioutil.ReadFile("mail.html")

	if err != nil {
		return fmt.Errorf("Error reading template: %w", err)
	}

	// Render the table template
	t := template.Must(template.New("template").Parse(string(emailTemplate)))
	err = t.Execute(renderedBody, data)

	if err != nil {
		return fmt.Errorf("Error rendering email body: %w", err)
	}

	message := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		renderedBody.String()

	err = smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(message))

	if err != nil {
		return fmt.Errorf("Error sending email: %w", err)
	}

	return nil

}
