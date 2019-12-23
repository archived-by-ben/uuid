package assets

// todo: format tables with go tab writer; add colour text

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Defacto2/uuid/v2/lib/database"
	"github.com/Defacto2/uuid/v2/lib/logs"

	"github.com/Defacto2/uuid/v2/lib/directories"

	"github.com/dustin/go-humanize"

	_ "github.com/go-sql-driver/mysql" // MySQL database driver
)

// files are unique UUID values used by the database and filenames
type files map[string]struct{}

type results struct {
	count int   // results handled
	fails int   // results that failed
	bytes int64 // bytes counted
}

type scan struct {
	path     string       // directory to scan
	output   string       // print output
	delete   bool         // delete any detected orphan files
	friendly bool         // humanize values shown by print output
	m        database.IDs // UUID values fetched from the database
}

var (
	empty  = database.Empty{}
	ignore files
	paths  []string // a collection of directories
	d      = directories.D
)

// AddTarFile saves the result of a fileWalk file into a TAR archive at path as the source file name.
// Source: cloudfoundry/archiver (https://github.com/cloudfoundry/archiver/blob/master/compressor/write_tar.go)
func AddTarFile(path, name string, tw *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	var link string
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
		if _, err = io.Copy(tw, file); err != nil {
			return err
		}
	}
	return nil
}

// Clean walks through and scans directories containing UUID files and erases any orphans that cannot be matched to the database
func Clean(delete bool, friendly bool, result string, target string) {
	output := result
	switch target {
	case "all":
		paths = append(paths, d.UUID, d.Emu, d.Backup, d.Img000, d.Img400, d.Img150)
	case "download":
		paths = append(paths, d.UUID, d.Backup)
	case "emulation":
		paths = append(paths, d.Emu)
	case "image":
		paths = append(paths, d.Img000, d.Img400, d.Img150)
	}
	// connect to the database
	rows, m := database.CreateUUIDMap()
	if console(output) {
		fmt.Printf("\nThe following files do not match any UUIDs in the database\n")
	}
	// parse directories
	var sum results
	for p := range paths {
		s := scan{path: paths[p], output: output, delete: delete, friendly: friendly, m: m}
		r, err := scanPath(s)
		if err != nil {
			if console(s.output) {
				fmt.Printf("Error: %v\n", err)
			} else {
				log.Printf("ERROR:%v\n", err)
			}
		}
		sum.bytes += r.bytes
		sum.count += r.count
		sum.fails += r.fails
	}
	// output a summary of the results
	if console(output) {
		fmt.Printf("\nTotal orphaned files discovered %v out of %v\n", humanize.Comma(int64(sum.count)), humanize.Comma(int64(rows)))
		if sum.fails > 0 {
			fmt.Printf("Due to errors, %v files could not be deleted\n", sum.fails)
		}
		if len(paths) > 1 {
			var pts string
			if friendly {
				pts = humanize.Bytes(uint64(sum.bytes))
			} else {
				pts = fmt.Sprintf("%v B", sum.bytes)
			}
			fmt.Printf("%v drive space consumed\n", pts)
		}
	}
}

// backup is used by scanPath to backup matched orphans
func backup(s *scan, list []os.FileInfo) {
	archive := make(map[string]struct{})
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
			archive[fn] = empty
		}
	}
	// identify which files should be backed up
	baks := make(map[string]string)
	baks[d.UUID] = "uuid"
	baks[d.Img150] = "img-150xthumbs"
	baks[d.Img400] = "img-400xthumbs"
	baks[d.Img000] = "img-captures"
	if _, ok := baks[s.path]; ok {
		t := time.Now()
		name := fmt.Sprintf("%vbak-%v-%v-%v-%v-%v%v%v.tar", d.Backup, baks[s.path], t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		basepath := s.path
		// create tar archive
		newTar, err := os.Create(name)
		logs.Check(err)
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
				if c == 1 && console(s.output) {
					fmt.Printf("Archiving these files before deletion\n\n")
				}
				return AddTarFile(path, name, tw)
			}
			return nil // no match
		})
		// if backup fails, then abort deletion
		if c == 0 || err != nil {
			// clean up any loose archives
			newTar.Close()
			err := os.Remove(name)
			logs.Check(err)
		}
	}
}

// console parses output flag to decide if to print to stdout
func console(flag string) bool {
	switch flag {
	case "none":
		return false
	default:
		return true
	}
}

// createDirectories generates a series of UUID subdirectories
func createDirectories() {
	createDirectory(d.Base)
	createDirectory(d.UUID)
	createDirectory(d.Emu)
	createDirectory(d.Backup)
	createDirectory(d.Img000)
	createDirectory(d.Img400)
	createDirectory(d.Img150)
}

// createDirectory creates a UUID subdirectory provided to path
func createDirectory(path string) bool {
	src, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			panic(err)
		}
		return true
	}
	if src.Mode().IsRegular() {
		fmt.Println(path, "already exist as a file!")
		return false
	}
	return false
}

// delete is used by scanPath to remove matched orphans
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
		uuid := strings.TrimSuffix(base, filepath.Ext(base))
		// search the map `m` for `UUID`, the result is saved as a boolean to `exists`
		_, exists := s.m[uuid]
		if !exists {
			r.count++
			r.bytes += file.Size()
			var del string
			if s.delete {
				del = fmt.Sprintf("  ✔")
				fn := fmt.Sprintf("%v%v", s.path, file.Name())
				rm := os.Remove(fn)
				if rm != nil {
					if console(s.output) {
						log.Printf("ERROR:%v\n", rm)
					} else {
						del = fmt.Sprintf("  ✖")
						r.fails++
					}
				}
			}
			if console(s.output) {
				var fs, mt string
				if s.friendly {
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

// ignoreList is used by scanPath to filter files that should not be erased
func ignoreList(path string) files {
	i := make(map[string]struct{})
	i["00000000-0000-0000-0000-000000000000"] = empty
	i["blank.png"] = empty
	if path == d.Emu {
		i["g_drive.zip"] = empty
		i["s_drive.zip"] = empty
		i["u_drive.zip"] = empty
		i["dosee-core.js"] = empty
		i["dosee-core.mem"] = empty
	}
	return i
}

// scanPath gets a list of filenames located in s.path and matches the results against the list generated by CreateUUIDMap
func scanPath(s scan) (results, error) {
	if console(s.output) {
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
	if console(s.output) {
		var dsc string
		if s.friendly {
			dsc = humanize.Bytes(uint64(r.bytes))
		} else {
			dsc = fmt.Sprintf("%v B", r.bytes)
		}
		fmt.Printf("\n%v orphaned files\n%v drive space consumed\n", r.count, dsc)
	}
	return r, nil // number of orphaned files discovered, deletion failures, their cumulative size in bytes
}
