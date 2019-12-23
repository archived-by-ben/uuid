package directories

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

// random characters used in randStringBytes()
const random = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321 .!?"

// Dir is a collection of paths containing files.
type Dir struct {
	Base   string // base directory path that hosts these other subdirectories
	UUID   string // path to file downloads with UUID as filenames
	Image  string // path to image previews and thumbnails
	File   string // path to webapp generated files such as JSON/XML
	Emu    string // path to the DOSee emulation files
	Backup string // path to the backup archives or previously removed files
	Img150 string // path to 150x150 squared thumbnails
	Img400 string // path to 400x400 squared thumbnails
	Img000 string // path to screencaptures and previews
}

var (
	// D are directory paths to UUID named files
	D = Dir{Base: "/Users/ben/Defacto2/", UUID: "uuid/", Image: "images/", File: "files/"}
)

// Init initializes the subdirectories and UUID structure
func Init(create bool) Dir {
	D.Emu = D.Base + D.File + "emularity.zip/"
	D.Backup = D.Base + D.File + "backups/"
	D.Img000 = D.Base + D.Image + "000x/"
	D.Img400 = D.Base + D.Image + "400x/"
	D.Img150 = D.Base + D.Image + "150x/"
	D.UUID = D.Base + D.UUID
	if create {
		createPlaceHolders()
	}
	return D
}

// createPlaceHolders generates a collection placeholder files in the UUID subdirectories
func createPlaceHolders() {
	createHolderFiles(D.UUID, 1000000, 9)
	createHolderFiles(D.Emu, 1000000, 2)
	createHolderFiles(D.Img000, 1000000, 9)
	createHolderFiles(D.Img400, 500000, 9)
	createHolderFiles(D.Img150, 100000, 9)
}

// createHolderFiles generates a number of placeholder files in the given directory
func createHolderFiles(dir string, size int, number uint) {
	if number > 9 {
		log.Fatalf("Invalid prefix %v, %v", number, fmt.Errorf("it must be between 0 and 9"))
	}
	var i uint
	for i = 0; i <= number; i++ {
		createHolderFile(dir, size, i)
	}
}

// createHolderFile generates a placeholder file filled with random text in the given directory,
// the size of the file determines the number of random characters and the prefix is a digit between
// 0 and 9 is appended to the filename
func createHolderFile(dir string, size int, prefix uint) {
	if prefix > 9 {
		log.Fatalf("Invalid prefix %v, %v", prefix, fmt.Errorf("it must be between 0 and 9"))
	}
	name := fmt.Sprintf("00000000-0000-0000-0000-00000000000%v", prefix)
	if _, err := os.Stat(dir + name); err == nil {
		return // don't overwrite existing files
	}
	rand.Seed(time.Now().UnixNano())
	text := []byte(randStringBytes(size))
	if err := ioutil.WriteFile(dir+name, text, 0644); err != nil {
		log.Fatal("Failed to write file", err)
	}
}

// randStringBytes generates a random string of n x characters
func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = random[rand.Int63()%int64(len(random))]
	}
	return string(b)
}
