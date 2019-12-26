package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Defacto2/uuid/v2/lib/archive"
	"github.com/Defacto2/uuid/v2/lib/directories"
	"github.com/Defacto2/uuid/v2/lib/logs"

	_ "github.com/go-sql-driver/mysql" // MySQL database driver
	"github.com/google/uuid"
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
	// TODO move to configuration
	d       = Connection{Name: "defacto2-inno", User: "root", Pass: "password", Server: "tcp(localhost:3306)"}
	proofID string
	pwPath  string // The path to a secured text file containing the d.User login password
)

func recordNew(values []sql.RawBytes) bool {
	if values[2] == nil || string(values[2]) != string(values[3]) {
		return false
	}
	return true
}

// CreateProof ...
func CreateProof(id string, ow bool, all bool) error {
	if !validUUID(id) && !validID(id) {
		return fmt.Errorf("invalid id given %q it needs to be an auto-generated MySQL id or an uuid", id)
	}
	proofID = id
	return CreateProofs(ow, all)
}

// CreateProofs is a placeholder to scan archives.
func CreateProofs(ow bool, all bool) error {
	db := Connect()
	defer db.Close()
	s := "SELECT `id`,`uuid`,`deletedat`,`createdat`,`filename`,`file_zip_content`,`updatedat`,`platform`"
	w := "WHERE `section` = 'releaseproof'"
	if proofID != "" {
		switch {
		case validUUID(proofID):
			w = fmt.Sprintf("%v AND `uuid`=%q", w, proofID)
		case validID(proofID):
			w = fmt.Sprintf("%v AND `id`=%q", w, proofID)
		}
	}
	rows, err := db.Query(s + "FROM `files`" + w)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
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
	// todo move to sep func to allow individual record parsing
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		logs.Check(err)
		if new := recordNew(values); new == false && all == false {
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
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			switch columns[i] {
			case "id":
				fmt.Printf("✓ item %v (%v) ", cnt, value)
			case "uuid":
				fmt.Printf("%v, ", value)
			case "createdat":
				fmt.Printf("%v, ", value)
			case "filename":
				fmt.Printf("%v\n", value)
			case "file_zip_content":
				if col == nil || ow {
					if u := fileZipContent(r); !u {
						continue
					}
					// todo: tag platform based on found files
					err := archive.Extract(r.File, r.UUID)
					logs.Log(err)
				}
			case "deletedat":
			case "updatedat": // ignore
			default:
				fmt.Printf("   %v: %v\n", columns[i], value)
			}
		}
		fmt.Println("---------------")
	}
	logs.Check(rows.Err())
	fmt.Println("Total proofs handled: ", cnt)
	if missing > 0 {
		fmt.Println("UUID files not found: ", missing)
	}
	return nil
}

func fileZipContent(r Record) bool {
	a, err := archive.Read(r.File)
	if err != nil {
		logs.Log(err)
		return false
	}
	Update(r.ID, strings.Join(a, "\n"))
	return true
}

// Update is a temp SQL update func.
func Update(id string, content string) {
	db := Connect()
	defer db.Close()
	update, err := db.Prepare("UPDATE files SET file_zip_content=?,updatedat=NOW(),updatedby=?,platform=?,deletedat=NULL,deletedby=NULL WHERE id=?")
	logs.Check(err)
	update.Exec(content, UpdateID, "image", id)
	fmt.Println("Updated file_zip_content")
}

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database.
func CreateUUIDMap() (int, IDs) {
	db := Connect()
	defer db.Close()
	// query database
	var id, uuid string
	rows, err := db.Query("SELECT `id`,`uuid` FROM `files`")
	logs.Check(err)
	m := IDs{} // this map is to store all the UUID values used in the database
	// handle query results
	rc := 0 // row count
	for rows.Next() {
		err = rows.Scan(&id, &uuid)
		logs.Check(err)
		m[uuid] = Empty{} // store record `uuid` value as a key name in the map `m` with an empty value
		rc++
	}
	return rc, m
}

// Connect to the database.
func Connect() *sql.DB {
	pw := readPassword()
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", d.User, pw, d.Server, d.Name))
	logs.Check(err)
	// ping the server to make sure the connection works
	err = db.Ping()
	logs.Check(err)
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
	logs.Check(err)
	return strings.TrimSpace(fmt.Sprintf("%s", pw))
}

func validUUID(id string) bool {
	if _, err := uuid.Parse(id); err != nil {
		return false
	}
	return true
}

func validID(id string) bool {
	if _, err := strconv.Atoi(id); err != nil {
		return false
	}
	return true
}
