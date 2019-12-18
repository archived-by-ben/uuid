package database

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	_ "github.com/go-sql-driver/mysql"
)

func TestCreateUUIDMap(t *testing.T) {

	// Creates sqlmock database connection and a mock to manage expectations.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	// Closes the database and prevents new queries from starting.
	defer db.Close()
	// Here we are creating rows in our mocked database.
	rows := sqlmock.NewRows([]string{"id", "uuid"}).
		AddRow(1, "00000000-0000-0000-0000-000000000000").
		AddRow(2, "10000000-0000-0000-0000-000000000000").
		AddRow(3, "20000000-0000-0000-0000-000000000000")

	// This is most important part in our test. Here, literally, we are altering SQL query from MenuByNameAndLanguage
	// function and replacing result with our expected result.
	mock.ExpectQuery("^SELECT (.+) FROM files*").
		WithArgs("main", "en").
		WillReturnRows(rows)

	tests := []struct {
		name  string
		want  int
		want1 IDs
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := CreateUUIDMap()
			if got != tt.want {
				t.Errorf("CreateUUIDMap() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("CreateUUIDMap() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_readPassword(t *testing.T) {

	// create a temporary file with the content EXAMPLE
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	text := []byte("EXAMPLE")
	if _, err = tmpFile.Write(text); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	tests := []struct {
		name   string
		pwPath string
		want   string
	}{
		{"empty", "", "password"},
		{"invalid", "/tmp/Test_readPassword/validfile", "password"},
		{"temp", tmpFile.Name(), "EXAMPLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwPath = tt.pwPath
			if got := readPassword(); got != tt.want {
				t.Errorf("readPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
