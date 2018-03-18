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
var seedData = []string{
	"1234567890",
	"123 456 7891",
	"(123) 456 7892",
	"(123) 456-7893",
	"123-456-7894",
	"123-456-7890",
	"1234567892",
	"(123)456-7892",
}

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

	rows, err := db.Query("SELECT phone FROM phonenumbers;")
	if err != nil {
		log.Fatalf("Error retrieving data.\n%s", err)
	}
	defer rows.Close()

	normalizedResults := make(normalizedNumbers)
	queryResults := []string{}
	for rows.Next() {
		var fetched string
		if err := rows.Scan(&fetched); err != nil {
			log.Fatalln(err)
		}
		queryResults = append(queryResults, fetched)
		normalizedResults[normalizePhoneNumber(fetched)] = true
	}

	if len(queryResults) == 0 {
		if err := seedDBWithPhoneNumbers(db, seedData...); err != nil {
			log.Printf("Error populating database.\n%+v", err)
		}
	}
	if err = deleteNotNormalsFromDB(db, normalizedResults, queryResults...); err != nil {
		log.Printf("%+v", err)
	}
	err = writeBackCorrectPhoneNumbers(db, normalizedResults)
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

type normalizedNumbers map[string]bool

func seedDBWithPhoneNumbers(db *sql.DB, args ...string) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			log.Printf("Error seeding database: %+v", err)
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	for _, value := range args {
		_, err := tx.Exec("INSERT INTO phonenumbers VALUES ($1);", value)
		if err != nil {
			return errors.Wrapf(err, "Failed to seed database with value %s", value)
		}
	}
	return nil
}

func writeBackCorrectPhoneNumbers(db *sql.DB, args normalizedNumbers) (err error) {
	for v := range args {
		_, err = db.Exec("INSERT INTO phonenumbers VALUES ($1);", v)
		if err != nil {
			log.Printf("Skipped writing %s to database: %s\n", v, err)
		}
	}
	return err
}

func deleteNotNormalsFromDB(db *sql.DB, normal normalizedNumbers, unNormalizedNumbers ...string) error {
	for _, v := range unNormalizedNumbers {
		if _, ok := normal[v]; !ok {
			_, err := db.Exec("DELETE FROM phonenumbers WHERE phone = $1;", v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
