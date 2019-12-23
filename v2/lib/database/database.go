package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Defacto2/uuid/v2/lib/archive"
	"github.com/Defacto2/uuid/v2/lib/directories"

	_ "github.com/go-sql-driver/mysql" // MySQL database driver
)

// UpdateID is a user id to use with the updatedby column.
const UpdateID string = "b66dc282-a029-4e99-85db-2cf2892fffcc"

// Connection information for a MySQL database.
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

// Record of a file item.
type Record struct {
	ID   string // mysql auto increment id
	UUID string // record unique id
	File string // absolute path to file
	Name string // filename
}

var (
	d      = Connection{Name: "defacto2-inno", User: "root", Pass: "password", Server: "tcp(localhost:3306)"}
	pwPath string // The path to a secured text file containing the d.User login password
)

func recordNew(values []sql.RawBytes) bool {
	if values[2] == nil || string(values[2]) != string(values[3]) {
		return false
	}
	return true
}

// CreateProof is a placeholder to scan archives.
func CreateProof() {
	db := connect()
	defer db.Close()
	s := "SELECT `id`,`uuid`,`deletedat`,`createdat`,`filename`,`file_zip_content`"
	w := "WHERE `section` = 'releaseproof'"
	rows, err := db.Query(s + "FROM `files`" + w)
	checkErr(err)
	columns, err := rows.Columns()
	checkErr(err)
	values := make([]sql.RawBytes, len(columns))
	// more information: https://github.com/go-sql-driver/mysql/wiki/Examples
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	dir := directories.Init(false)
	// fetch the rows
	cnt := 0
	missing := 0
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		checkErr(err)
		if new := recordNew(values); new == false {
			continue
		}
		cnt++
		r := Record{ID: string(values[0]), UUID: string(values[1]), Name: string(values[4])}
		r.File = filepath.Join(dir.UUID, r.UUID)
		// ping file
		if _, err := os.Stat(r.File); os.IsNotExist(err) {
			fmt.Printf("✗ item %v (%v) missing %v\n", cnt, r.ID, r.File)
			missing++
			continue
		}
		// iterate through each value
		var value string
		for i, col := range values {
			switch columns[i] {
			case "file_zip_content":
				if col != nil {
					if u := fileZipContent(r); !u {
						continue
					}
					archive.ExtractArchive(r.File, r.UUID)
				}
			case "deletedat", "updatedat": // ignore
			default:
				if col == nil {
					value = "NULL"
				} else {
					value = string(col)
				}
				fmt.Println(columns[i], ": ", value)
			}
		}
		fmt.Println("---------------")
	}
	checkErr(rows.Err())
	fmt.Println("Total proofs handled: ", cnt)
	if missing > 0 {
		fmt.Println("UUID files not found: ", missing)
	}
}

func fileZipContent(r Record) bool {
	fmt.Printf("archive %v content needs to be scanned\n", r.Name)
	a, err := archive.ReadArchive(r.File)
	if err != nil {
		log.Println(err)
		return false
	}
	Update(r.ID, strings.Join(a, "\n"))
	return true
}

// Update is a temp SQL update func.
func Update(id string, content string) {
	db := connect()
	defer db.Close()
	update, err := db.Prepare("UPDATE files SET file_zip_content=? WHERE id=?")
	checkErr(err)
	update.Exec(content, id)
	log.Println("Updated ID:", id, "file_zip_content =", content)
}

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database.
func CreateUUIDMap() (int, IDs) {
	db := connect()
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

// checkErr logs any errors.
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// connect to the database.
func connect() *sql.DB {
	pw := readPassword()
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", d.User, pw, d.Server, d.Name))
	checkErr(err)
	// ping the server to make sure the connection works
	err = db.Ping()
	checkErr(err)
	return db
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
