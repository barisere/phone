package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var (
	dbHost   string
	dbPort   int
	dbUser   string
	dbPasswd string
	dbName   string
)

func init() {
	flag.StringVar(&dbHost, "host", "localhost", "host name")
	flag.IntVar(&dbPort, "port", 5432, "connection port")
	flag.StringVar(&dbUser, "username", "", "connection username")
	flag.StringVar(&dbPasswd, "password", "", "connection password")
	flag.StringVar(&dbName, "database", "gophercises", "name of the database")
	flag.Parse()
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPasswd, dbName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to database.\n%s", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Unable to connect to database.\n%s", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT phone FROM phonenumbers")
	if err != nil {
		log.Fatalf("Error retrieving data.\n%s", err)
	}
	defer rows.Close()

	results := []queryResult{}
	for rows.Next() {
		var fetched string
		if err := rows.Scan(&fetched); err != nil {
			log.Fatalln(err)
		}
		results = append(results, queryResult{
			fetchedValue:    fetched,
			normalizedValue: normalizePhoneNumber(fetched)})
	}
	if len(results) == 0 {
		seedData := []string{
			"1234567890",
			"123 456 7891",
			"(123) 456 7892",
			"(123) 456-7893",
			"123-456-7894",
			"123-456-7890",
			"1234567892",
			"(123)456-7892",
		}
		if err := seedDBWithPhoneNumbers(db, seedData...); err != nil {
			log.Printf("Error populating database.\n%+v", err)
		}
	}
	err = writeBackCorrectPhoneNumbers(db, results...)
	if err != nil {
		log.Printf("%+v", err)
	}
}

var phoneNumberMatcher = regexp.MustCompile("[0-9]+")

func normalizePhoneNumber(phoneNumber string) string {
	normalized := strings.FieldsFunc(phoneNumber, func(r rune) bool {
		return !phoneNumberMatcher.MatchString(strconv.QuoteRune(r))
	})
	return strings.Join(normalized, "")
}

type queryResult struct {
	fetchedValue    string
	normalizedValue string
}

func seedDBWithPhoneNumbers(db *sql.DB, args ...string) (err error) {
	tx, err := db.Begin()
	defer func() {
		if err != nil {
			log.Printf("Error seeding database: %+v", err)
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	for _, value := range args {
		_, err := tx.Exec("INSERT INTO phonenumbers VALUES ($1)", value)
		if err != nil {
			return errors.Wrapf(err, "Failed to seed database with value %s", value)
		}
	}
	return nil
}

func writeBackCorrectPhoneNumbers(db *sql.DB, args ...queryResult) (err error) {
	for i := range args {
		if args[i].fetchedValue != args[i].normalizedValue {
			_, err := db.Exec("DELETE FROM phonenumbers WHERE phone = $1;", args[i].fetchedValue)
			if err != nil {
				return err
			}
			_, err = db.Exec("INSERT INTO phonenumbers VALUES ($1);", args[i].normalizedValue)
			if err != nil {
				log.Printf("Skipped writing %s to database: %s\n", args[i].normalizedValue, err)
			}
		}
	}
	return err
}
