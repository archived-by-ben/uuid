package database

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	unarr "github.com/gen2brain/go-unarr"

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

// IDs are unique UUID values used by the database and filenames
type IDs map[string]struct{}

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

var archives = []string{"zip", "tar.gz", "tar", "rar", "gz", "lzh", "lha", "cab", "arj", "arc", "7z"}

// ReadArchive returns a list of files within rar, tar, zip, 7z archives
func ReadArchive(name string) []string {
	a, err := unarr.NewArchive(name)
	checkErr(err)
	defer a.Close()
	list, err := a.List()
	checkErr(err)
	return list
}

// ExtractArchive decompresses and parses an archive
func ExtractArchive(name string) {
	// create temp dir
	tempDir, err := ioutil.TempDir("", "extarc-")
	checkErr(err)
	//defer os.RemoveAll(tempDir)
	// extract archive
	a, err := unarr.NewArchive(name)
	checkErr(err)
	defer a.Close()
	a.Extract(tempDir)
	// list temp dir
	fmt.Println("temp: ", tempDir)
	files, err := ioutil.ReadDir(tempDir)
	checkErr(err)
	for _, file := range files {
		fmt.Println("> ", file.Name(), os.PathSeparator)
		file := tempDir + "/" + file.Name()
		fmime, err := mimetype.DetectFile(file)
		checkErr(err)
		fmt.Println(">> ", fmime)
		switch fmime.Extension() {
		case ".jpg":
			// convert to PNG then resize then optimize
			// todo run 3rd party img optimizations
			MakeThumb(file, 400)
			MakeThumb(file, 150)
		case ".txt":
			fmt.Println("NFO!")
		default:
			fmt.Println(fmime.Extension())
		}
	}
}

// MakeThumb creates a thumb from an image that is size pixel in width and height
func MakeThumb(file string, size int) {
	cp := CopyFile(file, "_"+fmt.Sprintf("%v", size)+"x")
	fmt.Println("duplicated ", cp)
	src, err := imaging.Open(cp)
	checkErr(err)
	src = imaging.Resize(src, size, 0, imaging.Lanczos)
	src = imaging.CropAnchor(src, size, size, imaging.Center)
	err = imaging.Save(src, cp)
	checkErr(err)
}

// CopyFile duplicates a file and appends prefix to its filename
func CopyFile(name string, prefix string) string {
	src, err := os.Open(name)
	checkErr(err)
	defer src.Close()
	ext := filepath.Ext(name)
	fn := strings.TrimSuffix(name, ext)
	dest, err := os.OpenFile(fn+prefix+ext, os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer dest.Close()
	_, err = io.Copy(dest, src)
	checkErr(err)
	return fn + prefix + ext
}

// CreateProof is a placeholder to scan archives
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
	// fetch the rows
	cnt := 0
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		checkErr(err)
		if new := recordNew(values); new == false {
			continue
		}
		// iterate through each value
		var value string
		for i, col := range values {
			switch columns[i] {
			case "file_zip_content":
				// todo check filename extension
				if col == nil {
					fmt.Println("archive content needs to be scanned")
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
		cnt++
	}
	checkErr(rows.Err())
	fmt.Println("Total proofs handled, ", cnt)
}

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database
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

// CheckErr logs any errors
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// connect to the database
func connect() *sql.DB {
	pw := readPassword()
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", d.User, pw, d.Server, d.Name))
	checkErr(err)
	// ping the server to make sure the connection works
	err = db.Ping()
	checkErr(err)
	return db
}

// readPassword attempts to read and return the Defacto2 database user password when stored in a local text file
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
