// uuid.go - Defacto2 manager of UUID named files
// © Ben Garrett
// Partial © Cloud Foundry Foundation
//
// References:
// https://golang.org/pkg/os/#FileInfo
// Go by Example: Maps https://gobyexample.com/maps
// http://stackoverflow.com/questions/10485743/contains-method-for-a-slice
// Go: Slice search vs map lookup http://www.darkcoding.net/software/go-slice-search-vs-map-lookup/

package main

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

	docopt "github.com/docopt/docopt-go"
	humanize "github.com/dustin/go-humanize"
	_ "github.com/go-sql-driver/mysql"
)

const (
	version  string = "Defacto2 UUID Tool 1.1.0" // Application title and version
	dbServer string = "tcp(localhost:3306)"      // Database server connection, protocol (IP or domain address:port number)
)

var (
	dbName        string // Database name
	dbUser        string // Database username for login
	dbPass        string // Database username password for login [SHOULD BE LEFT BLANK]
	pwPath        string // The path to a secured text file containing the dbUser login password
	pathUUID      string // Path to file downloads named as UUID
	pathImageBase string // Path to image previews and thumbnails
	pathFilesBase string // Path to webapp generated files such as JSON/XML
	pathBackup    = fmt.Sprintf("%vbackups/", pathFilesBase)
	images150x    = fmt.Sprintf("%v150x150/", pathImageBase)
	images400x    = fmt.Sprintf("%v400x400/", pathImageBase)
	imagesDesc    = fmt.Sprintf("%vdescription/", pathImageBase)
	imagesInfo    = fmt.Sprintf("%vinformation/", pathImageBase)
	imagesPrev    = fmt.Sprintf("%vpreview/", pathImageBase)
	imagesCapt    = fmt.Sprintf("%vscreencapture/", pathImageBase)
	filesJSON     = fmt.Sprintf("%vjson/", pathFilesBase)
	filesEmu      = fmt.Sprintf("%vemularity/", pathFilesBase)
	filesEmuZip   = fmt.Sprintf("%vemularity.zip/", pathFilesBase)
	filesXMLOrg   = fmt.Sprintf("%vxml/organisation/", pathFilesBase) // TODO
)

// Empty is used as a blank value for search maps.
// See: https://dave.cheney.net/2014/03/25/the-empty-struct
type empty struct{}

func main() {
	// docopt CLI
	usage := `Defacto2 manager of UUID named files.

Usage:
  uuid all [options]
  uuid downloads [options]
  uuid emulation [text|zip] [options]
  uuid images [150 400 capt desc info prev] [options]
  uuid json [options]
  uuid -h | --help
  uuid --version

Options:
  -h --help        Show this screen.
  --version        Show version.
  
  --delete         Archive then delete all found orphaned files.
  --output=FORMAT  Output format (text|none) [Default: text]
  --raw            Dehumanize time and size details.`

	// parse CLI arguments to determine which directories to scan
	arguments, _ := docopt.Parse(usage, nil, true, version, false)
	var paths []string
	if arguments["downloads"] == true || arguments["all"] == true {
		paths = append(paths, pathUUID)
	}
	if arguments["emulation"] == true || arguments["all"] == true {
		if arguments["text"] == true {
			paths = append(paths, filesEmu)
		} else if arguments["zip"] == true {
			paths = append(paths, filesEmuZip)
		} else {
			paths = append(paths, filesEmu, filesEmuZip)
		}
	}
	if arguments["images"] == true || arguments["all"] == true {
		if arguments["all"] == true || (arguments["150"] == false && arguments["400"] == false &&
			arguments["capt"] == false && arguments["desc"] == false && arguments["info"] == false &&
			arguments["prev"] == false) {
			paths = append(paths, images150x, images400x, imagesCapt, imagesDesc, imagesInfo, imagesPrev)
		} else {
			if arguments["150"] == true {
				paths = append(paths, images150x)
			}
			if arguments["400"] == true {
				paths = append(paths, images400x)
			}
			if arguments["capt"] == true {
				paths = append(paths, imagesCapt)
			}
			if arguments["desc"] == true {
				paths = append(paths, imagesDesc)
			}
			if arguments["info"] == true {
				paths = append(paths, imagesInfo)
			}
			if arguments["prev"] == true {
				paths = append(paths, imagesPrev)
			}
		}
	}
	if arguments["json"] == true || arguments["all"] == true {
		paths = append(paths, filesJSON)
	}
	// CLI options
	var delete = arguments["--delete"].(bool) // Type assertions
	var output = arguments["--output"].(string)
	var rawData = arguments["--raw"].(bool)

	// connect to the database
	rows, m := createUUIDMap()
	if output != "none" {
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
			if rawData == false {
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
	if path == filesJSON {
		skip["file.list.json"] = empty{}
		skip["file.update.json"] = empty{}
		skip["organisation.list.json"] = empty{}
	} else if path == filesEmuZip {
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
	if delete == true {
		toArchive := make(map[string]struct{})
		for _, file := range files {
			if file.IsDir() == true {
				continue // ignore directories
			}
			if _, file := skip[file.Name()]; file == true {
				continue // ignore files
			}
			base := file.Name()
			UUID := strings.TrimSuffix(base, filepath.Ext(base))
			// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
			_, exists := m[UUID]
			if exists == false {
				toArchive[file.Name()] = empty{}
			}
		}
		// identify which files should be backed up
		tn := make(map[string]string)
		tn[pathUUID] = "uuid"
		tn[images150x] = "img-150xthumbs"
		tn[images400x] = "img-400xthumbs"
		tn[imagesCapt] = "img-captures"
		tn[imagesDesc] = "img-desc"
		tn[imagesInfo] = "img-info"
		tn[imagesPrev] = "img-prev"
		if _, ok := tn[path]; ok == true {
			t := time.Now()
			dest := fmt.Sprintf("%vbak-%v-%v-%v-%v-%v%v%v.tar", pathBackup, tn[path], t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
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
				if _, ok := toArchive[relative]; ok == true {
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
		if file.IsDir() == true {
			continue // ignore directories
		}
		if _, file := skip[file.Name()]; file == true {
			continue // ignore files
		}
		base := file.Name()
		UUID := strings.TrimSuffix(base, filepath.Ext(base))
		// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
		_, exists := m[UUID]
		if exists == false {
			cnt++
			ts += file.Size()
			var del string
			if delete == true {
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
				if rawData == false {
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
		if rawData == false {
			dsc = humanize.Bytes(uint64(ts))
		} else {
			dsc = fmt.Sprintf("%v B", ts)
		}
		fmt.Printf("\n%v orphaned files\n%v drive space consumed\n", cnt, dsc)
	}
	return cnt, fails, ts // number of orphaned files discovered, deletion failures, their cumulative size in bytes
}

// ReadPassword attempts to read and return the Defacto2 database user password when stored in a local text file.
func readPassword() string {
	// fetch database password
	pwFile, err := os.Open(pwPath)
	// return an empty password if path fails
	if err != nil {
		log.Print("WARNING:", err)
		return dbPass
	}
	defer pwFile.Close()
	pw, err := ioutil.ReadAll(pwFile)
	checkErr(err)
	return strings.TrimSpace(fmt.Sprintf("%s", pw))
}

// CreateUUIDMap builds a map of all the unique UUID values stored in the Defacto2 database.
func createUUIDMap() (int, map[string]struct{}) {
	// fetch database password
	password := readPassword()

	// connect to the database
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@%v/%v", dbUser, password, dbServer, dbName))
	checkErr(err)

	// query database
	var id, uuid string
	rows, err := db.Query("SELECT `id`,`uuid` FROM `files`")
	checkErr(err)

	defer db.Close()

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

func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}
