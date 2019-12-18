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
	var sum results
	for p := range paths {
		s := scan{path: paths[p], output: output, delete: delete, rawData: rawData, m: m}
		r, err := scanPath(s)
		if err != nil {
			if s.output == "none" {
				log.Printf("ERROR:%v\n", err)
			} else {
				fmt.Printf("Error: %v\n", err)
			}
		}
		sum.bytes += r.bytes
		sum.count += r.count
		sum.fails += r.fails
	}
	// output a summary of the results
	if output != "none" {
		fmt.Printf("\nTotal orphaned files discovered %v out of %v\n", humanize.Comma(int64(sum.count)), humanize.Comma(int64(rows)))
		if sum.fails > 0 {
			fmt.Printf("Due to errors, %v files could not be deleted\n", sum.fails)
		}
		if len(paths) > 1 {
			var pts string
			if !rawData {
				pts = humanize.Bytes(uint64(sum.bytes))
			} else {
				pts = fmt.Sprintf("%v B", sum.bytes)
			}
			fmt.Printf("%v drive space consumed\n", pts)
		}
	}
}

type results struct {
	count int
	fails int
	bytes int64
}

type scan struct {
	path    string
	output  string
	delete  bool
	rawData bool
	m       files
}

type files map[string]struct{}

var ignore files

func ignoreList(path string) files {
	i := make(map[string]struct{})
	i["00000000-0000-0000-0000-000000000000"] = empty{}
	i["blank.png"] = empty{}
	if path == p.emu {
		i["g_drive.zip"] = empty{}
		i["s_drive.zip"] = empty{}
		i["u_drive.zip"] = empty{}
		i["dosee-core.js"] = empty{}
		i["dosee-core.mem"] = empty{}
	}
	return i
}

func backup(s *scan, list []os.FileInfo) {
	var archive files
	for _, file := range list {
		if file.IsDir() {
			continue // ignore directories
		}
		if _, file := ignore[file.Name()]; file {
			continue // ignore files
		}
		fn := file.Name()
		id := strings.TrimSuffix(fn, filepath.Ext(fn))
		// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
		_, exists := s.m[id]
		if !exists {
			archive[file.Name()] = empty{}
		}
	}
	// identify which files should be backed up
	baks := make(map[string]string)
	baks[p.uuid] = "uuid"
	baks[p.img150] = "img-150xthumbs"
	baks[p.img400] = "img-400xthumbs"
	baks[p.img000] = "img-captures"
	if _, ok := baks[s.path]; ok {
		t := time.Now()
		name := fmt.Sprintf("%vbak-%v-%v-%v-%v-%v%v%v.tar", p.backup, baks[s.path], t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		basepath := s.path
		// create tar archive
		newTar, err := os.Create(name)
		checkErr(err)
		tw := tar.NewWriter(newTar)
		defer tw.Close()
		c := 0
		// walk through `path` and match any files marked for deletion
		// Partial source: https://github.com/cloudfoundry/archiver/blob/master/compressor/write_tar.go
		err = filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			var name string
			if os.IsPathSeparator(path[len(path)-1]) {
				name, err = filepath.Rel(basepath, path)
			} else {
				name, err = filepath.Rel(filepath.Dir(basepath), path)
			}
			name = filepath.ToSlash(name)
			if err != nil {
				return err
			}
			if _, ok := archive[name]; ok {
				c++
				if c == 1 && s.output != "none" {
					fmt.Printf("Archiving these files before deletion\n\n")
				}
				return addTarFile(path, name, tw)
			}
			return nil // no match
		})
		// if backup fails, then abort deletion
		if c == 0 || err != nil {
			// clean up any loose archives
			newTar.Close()
			rm := os.Remove(name)
			checkErr(err)
			checkErr(rm)
		}
	}
}

func delete(s *scan, list []os.FileInfo) results {
	var r = results{count: 0, fails: 0, bytes: 0}
	for _, file := range list {
		if file.IsDir() {
			continue // ignore directories
		}
		if _, file := ignore[file.Name()]; file {
			continue // ignore files
		}
		base := file.Name()
		UUID := strings.TrimSuffix(base, filepath.Ext(base))
		// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
		_, exists := s.m[UUID]
		if !exists {
			r.count++
			r.bytes += file.Size()
			var del string
			if s.delete {
				del = fmt.Sprintf("  ✔")
				fn := fmt.Sprintf("%v%v", s.path, file.Name())
				rm := os.Remove(fn)
				if rm != nil {
					if s.output == "none" {
						log.Printf("ERROR:%v\n", rm)
					} else {
						del = fmt.Sprintf("  ✖")
						r.fails++
					}
				}
			}
			if s.output != "none" {
				var fs, mt string
				if !s.rawData {
					fs = humanize.Bytes(uint64(file.Size()))
					mt = file.ModTime().Format("2006-Jan-02 15:04:05")
				} else {
					fs = fmt.Sprint(file.Size())
					mt = fmt.Sprint(file.ModTime())
				}
				fmt.Printf("%v.\t%-44s\t%v\t%v  %v%v\n", r.count, base, fs, file.Mode(), mt, del)
			}
		}
	}
	return r
}

// ScanPath gets a list of filenames located in `path` and matches the results against the list generated by createUUIDMap.
func scanPath(s scan) (results, error) {
	if s.output != "none" {
		fmt.Printf("\nResults from %v\n\n", s.path)
	}
	// query file system
	list, err := ioutil.ReadDir(s.path)
	if err != nil {
		return results{}, err
	}
	// files to ignore
	ignore = ignoreList(s.path)
	// archive files that are to be deleted
	if s.delete {
		backup(&s, list)
	}
	// list and if requested, delete orphaned files
	r := delete(&s, list)
	if s.output != "none" {
		var dsc string
		if !s.rawData {
			dsc = humanize.Bytes(uint64(r.bytes))
		} else {
			dsc = fmt.Sprintf("%v B", r.bytes)
		}
		fmt.Printf("\n%v orphaned files\n%v drive space consumed\n", r.count, dsc)
	}
	return r, nil // number of orphaned files discovered, deletion failures, their cumulative size in bytes
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
