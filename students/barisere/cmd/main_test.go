package main

import (
	"database/sql"
	"fmt"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestNormalizePhoneNumber(t *testing.T) {
	testData := []struct {
		message  string
		data     string
		expected string
	}{
		{"A phone number in correct form should not be changed", "1234567890", "1234567890"},
		{"Spaces should be removed from phone numbers", "123 456 7891", "1234567891"},
		{"Parentheses should be removed from phone numbers", "(123) 456 7892", "1234567892"},
		{"Hyphens should be removed from phone numbers", "(123) 456-7893", "1234567893"},
	}
	for _, data := range testData {
		t.Run(data.message, func(t *testing.T) {
			if got := normalizePhoneNumber(data.data); got != data.expected {
				t.Errorf("Expected %s, got %s\n", data.expected, got)
			}
		})
	}
}

var (
	columns = []string{"phone"}
)

func TestWriteBackCorrectPhoneNumbers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Failed to create DB/mock:\n%v+", err)
	}
	defer db.Close()
	result := make(normalizedNumbers)
	for _, v := range seedData {
		result[normalizePhoneNumber(v)] = true
	}
	for v := range result {
		mock.ExpectExec("INSERT INTO phonenumbers VALUES").
			WithArgs(v).
			WillReturnResult(sqlmock.NewResult(1, 1)).
			WillReturnError(nil)
	}
	mock.MatchExpectationsInOrder(false)
	if err = writeBackCorrectPhoneNumbers(db, result); err != nil {
		t.Errorf("Writing back data failed with error:\n%+v", err)
	}
	for v := range result {
		t.Run(fmt.Sprintf("Should see %s in DB", v), func(t *testing.T) {
			if ok, err := seenInDB(db, mock, v); !ok || (err != nil) {
				t.Error(err)
			}
		})
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestSeedDBWithPhoneNumbers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Failed to create DB/mock:\n%v+", err)
	}
	defer db.Close()
	mock.ExpectBegin().WillReturnError(nil)
	for _, v := range seedData {
		mock.ExpectExec("^INSERT INTO phonenumbers").WithArgs(v).
			WillReturnError(nil).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit().WillReturnError(nil)
	err = seedDBWithPhoneNumbers(db, seedData...)
	if err != nil {
		t.Errorf("Seeding database failed:\n%+v", err)
	}

	for _, v := range seedData {
		t.Run(fmt.Sprintf("Should see %s in DB", v), func(t *testing.T) {
			if ok, err := seenInDB(db, mock, v); !ok || (err != nil) {
				t.Error(err)
			}
		})
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations:\n%+v", err)
	}
}

func TestDeleteNotNormalsFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Failed to create DB/mock:\n%v+", err)
	}
	defer db.Close()

	normalNumbers := make(normalizedNumbers)
	for _, v := range seedData {
		normalNumbers[normalizePhoneNumber(v)] = true
	}

	mock.ExpectBegin().WillReturnError(nil)
	for _, v := range seedData {
		mock.ExpectExec("^INSERT INTO phonenumbers").WithArgs(v).
			WillReturnError(nil).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit().WillReturnError(nil)
	err = seedDBWithPhoneNumbers(db, seedData...)
	if err != nil {
		t.Errorf("Seeding database failed:\n%+v", err)
	}

	for _, v := range seedData {
		if _, ok := normalNumbers[v]; !ok {
			mock.ExpectExec("DELETE FROM phonenumbers").WithArgs(v).
				WillReturnError(nil).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
	}
	if err = deleteNotNormalsFromDB(db, normalNumbers, seedData...); err != nil {
		t.Errorf("Deleting not normals from database failed:\n%+v", err)
	}
	for _, v := range seedData {
		if _, ok := normalNumbers[v]; !ok {
			t.Run(fmt.Sprintf("Should not see %s in DB", v), func(t *testing.T) {
				if ok, err := notSeenInDB(db, mock, v); !ok || (err != nil) {
					t.Error(err)
				}
			})
		}
	}
}

func seenInDB(db *sql.DB, mock sqlmock.Sqlmock, v string) (bool, error) {
	mock.ExpectQuery("SELECT phone FROM phonenumbers").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(v))
	row := db.QueryRow("SELECT phone FROM phonenumbers WHERE phone = $1", v)
	if row == nil {
		return false, fmt.Errorf("Did not see %s in database", v)
	}
	var result string
	if err := row.Scan(&result); err != nil {
		return false, fmt.Errorf("Error retrieving %s\n%+v", v, err)
	}
	if result != v {
		return false, fmt.Errorf("Unexpected value: want %s, got %s", v, result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		return false, fmt.Errorf("Unmet expectations:\n%+v", err)
	}
	return true, nil
}

func notSeenInDB(db *sql.DB, mock sqlmock.Sqlmock, v string) (bool, error) {
	seen, _ := seenInDB(db, mock, v)
	if seen {
		return !seen, fmt.Errorf("Saw %s in DB", v)
	}
	return !seen, nil
}
