package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	// MySQL database driver
	_ "github.com/go-sql-driver/mysql"
)

// Connection information for a MySQL database
type Connection struct {
	Name   string // database name
	User   string // access username
	Pass   string // access password
	Server string // host server protocol, address and port
}

// Empty is used as a blank value for search maps.
// See: https://dave.cheney.net/2014/03/25/the-empty-struct
type Empty struct{}

// IDs are unique UUID values used by the database and filenames.
type IDs map[string]struct{}

var (
	d      = Connection{Name: "defacto2-inno", User: "root", Pass: "password", Server: "tcp(localhost:3306)"}
	pwPath string // The path to a secured text file containing the d.User login password
)

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database.
func CreateUUIDMap() (int, IDs) {
	pw := readPassword()

	// connect to the database
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", d.User, pw, d.Server, d.Name))
	checkErr(err)
	defer db.Close()

	// query database
	var id, uuid string
	rows, err := db.Query("SELECT `id`,`uuid` FROM `files`")
	checkErr(err)

	m := IDs{} // this map is to store all the UUID values used in the database

	// handle query results
	rc := 0 // row count
	for rows.Next() {
		err = rows.Scan(&id, &uuid)
		checkErr(err)
		m[uuid] = Empty{} // store record `uuid` value as a key name in the map `m` with an empty value
		rc++
	}
	return rc, m
}

// CheckErr logs any errors
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// readPassword attempts to read and return the Defacto2 database user password when stored in a local text file.
func readPassword() string {
	// fetch database password
	pwFile, err := os.Open(pwPath)
	// return an empty password if path fails
	if err != nil {
		//log.Print("WARNING: ", err)
		return d.Pass
	}
	defer pwFile.Close()
	pw, err := ioutil.ReadAll(pwFile)
	checkErr(err)
	return strings.TrimSpace(fmt.Sprintf("%s", pw))
}
