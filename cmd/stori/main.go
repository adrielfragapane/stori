package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gocarina/gocsv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.S3Event) error {

	sess := session.Must(session.NewSession())

	for _, record := range event.Records {
		event := record.S3

		bucketName := event.Bucket.Name
		objectKey := event.Object.Key

		file, err := readFile(sess, bucketName, objectKey)
		if err != nil {
			return fmt.Errorf("Error reading file from S3: %w", err)
		}

		templateData, err := processCSV(file)

		template, err := readFile(sess, bucketName, "mail.html")
		if err != nil {
			return fmt.Errorf("Error reading template from S3: %w", err)
		}

		err = sendEmail(template, templateData)

		if err != nil {
			return fmt.Errorf("Error sending email: %w", err)
			// Helloo
		}
	}

	return nil
}

func readFile(sess *session.Session, bucketName string, objectKey string) (io.ReadCloser, error) {
	svc := s3.New(sess)

	params := &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	}

	resp, err := svc.GetObject(params)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func processCSV(csvFile io.ReadCloser) (TemplateData, error) {

	defer csvFile.Close()

	transactions := []Transaction{}

	if err := gocsv.Unmarshal(csvFile, &transactions); err != nil {
		return TemplateData{}, fmt.Errorf("Error parsing file: %w", err)
	}

	repository := newRepository()

	totalBalance := 0.0
	creditBalance := map[int]float64{}
	debitBalance := map[int]float64{}
	transactionsByMonth := map[int]int{}

	for _, transaction := range transactions {

		repository.saveTransaction(transaction)

		month, _ := strconv.Atoi(strings.Split(transaction.Date, "/")[0])

		totalBalance += transaction.Value

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

	for _, month := range sortedMonths {
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

func sendEmail(emailTemplate io.ReadCloser, data TemplateData) error {

	from := os.Getenv("EMAIL_USER")
	pass := os.Getenv("EMAIL_PASS")
	to := os.Getenv("EMAIL_TO")
	subject := "Account balance"

	renderedBody := new(bytes.Buffer)
	templateBody := new(bytes.Buffer)

	templateBody.ReadFrom(emailTemplate)

	// Render the table template
	t := template.Must(template.New("template").Parse(string(templateBody.String())))
	err := t.Execute(renderedBody, data)

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

type TemplateData struct {
	Rows    []MonthData
	Balance float64
}

type Transaction struct {
	Id    int     `csv:"Id" gorm:"column:id"`
	Date  string  `csv:"Date" gorm:"column:date"`
	Value float64 `csv:"Transaction" gorm:"column:value"`
}

func (Transaction) TableName() string {
	return "transactions"
}

type Account struct {
	Id      int     `gorm:"column:id"`
	Name    string  `gorm:"column:name"`
	Balance float64 `gorm:"column:balance"`
}

func (Account) TableName() string {
	return "account"
}

type MonthData struct {
	Month        string
	Transactions int
	DebitAmount  float64
	CreditAmount float64
}

type Repository interface {
	saveTransaction(transaction Transaction) error
}

type repository struct {
	db *gorm.DB
}

func newRepository() Repository {

	username := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", username, password, host, port, dbName),
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Transaction{})

	return &repository{db: db}
}

func (repo *repository) saveTransaction(transaction Transaction) error {

	result := repo.db.Create(&transaction)
	return result.Error
}
