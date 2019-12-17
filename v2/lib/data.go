package data

import (
	"archive/tar"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	// MySQL database driver
	"github.com/dustin/go-humanize"
	// MySQL database driver
	_ "github.com/go-sql-driver/mysql"
)

// Empty is used as a blank value for search maps.
// See: https://dave.cheney.net/2014/03/25/the-empty-struct
type empty struct{}

type db struct {
	name   string // database name
	user   string // access username
	pass   string // access password
	server string // server address
}

// Dir is a collection of paths containing files
type Dir struct {
	root   string
	uuid   string // path to file downloads named as UUID
	image  string // path to image previews and thumbnails
	file   string // path to webapp generated files such as JSON/XML
	emu    string
	backup string
	img150 string
	img400 string
	img000 string
}

var (
	d      = db{name: "defacto2-inno", user: "root", pass: "password", server: "tcp(localhost:3306)"}
	p      = Dir{root: "/Users/ben/Defacto2/", uuid: "uuid/", image: "images/", file: "files/"}
	paths  []string
	pwPath string // The path to a secured text file containing the dbUser login password
)

// Init xx
func Init() {
	p.emu = p.root + p.file + "emularity.zip/"
	p.backup = p.root + p.file + "backups/"
	p.img000 = p.root + p.image + "000x/"
	p.img400 = p.root + p.image + "400x/"
	p.img150 = p.root + p.image + "150x/"
	p.uuid = p.root + p.uuid
	//
	createDirectory(p.root)
	createDirectory(p.uuid)
	createDirectory(p.emu)
	createDirectory(p.backup)
	createDirectory(p.img000)
	createDirectory(p.img400)
	createDirectory(p.img150)
}

func createDirectory(dirName string) bool {
	src, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirName, 0755)
		if errDir != nil {
			panic(err)
		}
		return true
	}

	if src.Mode().IsRegular() {
		fmt.Println(dirName, "already exist as a file!")
		return false
	}

	return false
}

// Clean xxx
func Clean() {
	output := ""
	outArg := false
	delete := false
	rawData := false

	// paths
	paths = append(paths, p.uuid, p.emu, p.backup, p.img000, p.img400, p.img150)

	fmt.Printf("\npath: %v\np: %v\n", paths, p.uuid)

	// connect to the database
	rows, m := CreateUUIDMap()
	if outArg && output != "none" {
		fmt.Printf("\nThe following files do not match any UUIDs in the database\n")
	}
	// parse directories
	var tCount, tFails, count, fails int
	var tBytes, bytes int64
	for p := range paths {
		count, fails, bytes = scanPath(paths[p], output, delete, rawData, m)
		tBytes += bytes
		tCount += count
		tFails += fails
	}
	// output a summary of the results
	if output != "none" {
		fmt.Printf("\nTotal orphaned files discovered %v out of %v\n", humanize.Comma(int64(tCount)), humanize.Comma(int64(rows)))
		if tFails > 0 {
			fmt.Printf("Due to errors, %v files could not be deleted\n", tFails)
		}
		if len(paths) > 1 {
			var pts string
			if !rawData {
				pts = humanize.Bytes(uint64(tBytes))
			} else {
				pts = fmt.Sprintf("%v B", tBytes)
			}
			fmt.Printf("%v drive space consumed\n", pts)
		}
	}
}

// ScanPath gets a list of filenames located in `path` and matches the results against the list generated by createUUIDMap.
func scanPath(path string, output string, delete bool, rawData bool, m map[string]struct{}) (int, int, int64) {
	if output != "none" {
		fmt.Printf("\nResults from %v\n\n", path)
	}
	// query file system
	files, err := ioutil.ReadDir(path)
	if err != nil {
		if output == "none" {
			log.Printf("ERROR:%v\n", err)
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		return 0, 0, 0
	}
	// files to ignore
	skip := make(map[string]struct{})
	skip["00000000-0000-0000-0000-000000000000"] = empty{}
	skip["blank.png"] = empty{}
	if path == p.emu {
		skip["g_drive.zip"] = empty{}
		skip["s_drive.zip"] = empty{}
		skip["u_drive.zip"] = empty{}
		skip["dosee-core.js"] = empty{}
		skip["dosee-core.mem"] = empty{}
	}

	// handle found file results
	var ts int64
	cnt := 0
	// archive files that are to be deleted
	if delete {
		toArchive := make(map[string]struct{})
		for _, file := range files {
			if file.IsDir() {
				continue // ignore directories
			}
			if _, file := skip[file.Name()]; file {
				continue // ignore files
			}
			base := file.Name()
			UUID := strings.TrimSuffix(base, filepath.Ext(base))
			// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
			_, exists := m[UUID]
			if !exists {
				toArchive[file.Name()] = empty{}
			}
		}
		// identify which files should be backed up
		tn := make(map[string]string)
		tn[p.uuid] = "uuid"
		tn[p.img150] = "img-150xthumbs"
		tn[p.img400] = "img-400xthumbs"
		tn[p.img000] = "img-captures"
		if _, ok := tn[path]; ok {
			t := time.Now()
			dest := fmt.Sprintf("%vbak-%v-%v-%v-%v-%v%v%v.tar", p.backup, tn[path], t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
			absPath := path
			// create tar archive
			fileBak, err := os.Create(dest)
			checkErr(err)
			tw := tar.NewWriter(fileBak)
			defer tw.Close()
			c := 0
			// walk through `path` and match any files marked for deletion
			// Partial source: https://github.com/cloudfoundry/archiver/blob/master/compressor/write_tar.go
			err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				var relative string
				if os.IsPathSeparator(path[len(path)-1]) {
					relative, err = filepath.Rel(absPath, path)
				} else {
					relative, err = filepath.Rel(filepath.Dir(absPath), path)
				}
				relative = filepath.ToSlash(relative)
				if err != nil {
					return err
				}
				if _, ok := toArchive[relative]; ok {
					c++
					if c == 1 {
						if output != "none" {
							fmt.Printf("Archiving these files before deletion\n\n")
						}
					}
					return addTarFile(path, relative, tw)
				}
				return nil // no match
			})
			// if backup fails, then abort deletion
			if c == 0 || err != nil {
				// clean up any loose archives
				fileBak.Close()
				rm := os.Remove(dest)
				checkErr(err)
				checkErr(rm)
			}
		}
	}
	// list and if requested, delete orphaned files
	fails := 0
	for _, file := range files {
		if file.IsDir() {
			continue // ignore directories
		}
		if _, file := skip[file.Name()]; file {
			continue // ignore files
		}
		base := file.Name()
		UUID := strings.TrimSuffix(base, filepath.Ext(base))
		// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
		_, exists := m[UUID]
		if !exists {
			cnt++
			ts += file.Size()
			var del string
			if delete {
				del = fmt.Sprintf("  ✔")
				fn := fmt.Sprintf("%v%v", path, file.Name())
				rm := os.Remove(fn)
				if rm != nil {
					if output == "none" {
						log.Printf("ERROR:%v\n", rm)
					} else {
						del = fmt.Sprintf("  ✖")
						fails++
					}
				}
			}
			if output != "none" {
				var fs, mt string
				if !rawData {
					fs = humanize.Bytes(uint64(file.Size()))
					mt = file.ModTime().Format("2006-Jan-02 15:04:05")
				} else {
					fs = fmt.Sprint(file.Size())
					mt = fmt.Sprint(file.ModTime())
				}
				fmt.Printf("%v.\t%-44s\t%v\t%v  %v%v\n", cnt, base, fs, file.Mode(), mt, del)
			}
		}
	}
	if output != "none" {
		var dsc string
		if !rawData {
			dsc = humanize.Bytes(uint64(ts))
		} else {
			dsc = fmt.Sprintf("%v B", ts)
		}
		fmt.Printf("\n%v orphaned files\n%v drive space consumed\n", cnt, dsc)
	}
	return cnt, fails, ts // number of orphaned files discovered, deletion failures, their cumulative size in bytes
}

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database.
func CreateUUIDMap() (int, map[string]struct{}) {
	// fetch database password
	password := readPassword()

	// connect to the database
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", d.user, password, d.server, d.name))
	checkErr(err)
	defer db.Close()

	// query database
	var id, uuid string
	rows, err := db.Query("SELECT `id`,`uuid` FROM `files`")
	checkErr(err)

	m := make(map[string]struct{}) // this map is to store all the UUID values used in the database

	// handle query results
	rc := 0 // row count
	for rows.Next() {
		err = rows.Scan(&id, &uuid)
		checkErr(err)
		m[uuid] = empty{} // store record `uuid` value as a key name in the map `m` with an empty value
		rc++
	}
	return rc, m
}

// ReadPassword attempts to read and return the Defacto2 database user password when stored in a local text file.
func readPassword() string {
	// fetch database password
	pwFile, err := os.Open(pwPath)
	// return an empty password if path fails
	if err != nil {
		//log.Print("WARNING: ", err)
		return d.pass
	}
	defer pwFile.Close()
	pw, err := ioutil.ReadAll(pwFile)
	checkErr(err)
	return strings.TrimSpace(fmt.Sprintf("%s", pw))
}

// CheckErr logs any errors
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// Source: cloudfoundry/archiver
// https://github.com/cloudfoundry/archiver/blob/master/compressor/write_tar.go
func addTarFile(path, name string, tw *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	link := ""
	if fi.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(path); err != nil {
			return err
		}
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}

	if fi.IsDir() && !os.IsPathSeparator(name[len(name)-1]) {
		name = name + "/"
	}

	if hdr.Typeflag == tar.TypeReg && name == "." {
		// archiving a single file
		hdr.Name = filepath.ToSlash(filepath.Base(path))
	} else {
		hdr.Name = filepath.ToSlash(name)
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if hdr.Typeflag == tar.TypeReg {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}
