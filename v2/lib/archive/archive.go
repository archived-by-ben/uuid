package archive

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // register GIF decoding
	_ "image/jpeg" // register Jpeg decoding
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	unarr "github.com/gen2brain/go-unarr"
	"github.com/nickalie/go-webpbin"
	"github.com/yusukebe/go-pngquant"
	_ "golang.org/x/image/bmp"  // register BMP decoding
	_ "golang.org/x/image/tiff" // register TIFF decoding
	_ "golang.org/x/image/webp" // register WebP decoding
)

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

type task struct {
	name string // filename
	size int64  // file size
	cont bool   // continue, don't scan anymore images
}

// ExtractArchive decompresses and parses an archive
func ExtractArchive(name string) {
	// create temp dir
	tempDir, err := ioutil.TempDir("", "extarc-")
	checkErr(err)
	defer os.RemoveAll(tempDir)
	// extract archive
	a, err := unarr.NewArchive(name)
	checkErr(err)
	defer a.Close()
	a.Extract(tempDir)
	// list temp dir
	fmt.Println("temp: ", tempDir)
	files, err := ioutil.ReadDir(tempDir)
	checkErr(err)
	//t := tasks{thumb: false, text: false}
	th := task{name: "", size: 0, cont: false}
	tx := task{name: "", size: 0, cont: false}
	for _, file := range files {
		if th.cont && tx.cont {
			break
		}
		fn := path.Join(tempDir, file.Name())
		fmime, err := mimetype.DetectFile(fn)
		checkErr(err)
		fmt.Println(">", file.Name(), humanize.Bytes(uint64(file.Size())), fmime)
		switch fmime.Extension() {
		case ".bmp", ".gif", ".jpg", ".png", ".tiff", ".webp":
			if th.cont {
				continue
			}
			switch {
			case file.Size() > th.size:
				th.name = fn
				th.size = file.Size()
			}
		case ".txt":
			if tx.cont {
				continue
			}
			// todo copy file
			tx.name = fn
			tx.size = file.Size()
			tx.cont = true
		default:
			//fmt.Println(fmime.Extension())
		}
	}
	if n := th.name; n != "" {
		tImages(n)
	}
	dir(tempDir)
}

func tImages(n string) {
	// make these 4 image tasks multithread
	c := make(chan bool)
	go func() { ToPng(n, NewExt(n, ".png"), 1500); c <- true }()
	go func() { ToWebp(n, NewExt(n, ".webp")); c <- true }()
	go func() { MakeThumb(n, 400); c <- true }()
	go func() { MakeThumb(n, 150); c <- true }()
	_, _, _, _ = <-c, <-c, <-c, <-c // sync 4 tasks
	os.Remove(n)
}

// ToWebp converts any support format to a WebP image using a 3rd party library.
func ToWebp(src string, dest string) {
	// skip if already a webp image
	if m, _ := mimetype.DetectFile(src); m.Extension() == ".webp" {
		return
	}
	fmt.Println("Converting to WebP")
	err := webpbin.NewCWebP().
		Quality(70).
		InputFile(src).
		OutputFile(dest).
		Run()
	checkErr(err)
}

// ToPng converts any supported format to a compressed PNG image.
// helpful: https://www.programming-books.io/essential/go/images-png-jpeg-bmp-tiff-webp-vp8-gif-c84a45304ec3498081c67aa1ea0d9c49
func ToPng(src string, dest string, maxDimension int) {
	in, err := os.Open(src)
	checkErr(err)
	defer in.Close()
	// image.Decode will determine the format
	img, ext, err := image.Decode(in)
	checkErr(err)
	// inf, err := in.Stat()
	// checkErr(err)
	// fmt.Println("INF:", inf.Size())
	fmt.Printf("Converting %v to a compressed PNG\n", ext)
	// cap image size
	if maxDimension > 0 {
		fmt.Println("Resizing image down to", maxDimension, "pixels")
		img = imaging.Thumbnail(img, maxDimension, maxDimension, imaging.Lanczos)
	}
	// use the 3rd party CLI tool, pngquant to compress the PNG data
	img, err = pngquant.Compress(img, "4")
	checkErr(err)
	// adjust any configs to the PNG image encoder
	cfg := png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	// write the PNG data to img
	buf := new(bytes.Buffer)
	err = cfg.Encode(buf, img)
	checkErr(err)
	// save the PNG to a file
	out, err := os.Create(dest)
	checkErr(err)
	defer out.Close()
	buf.WriteTo(out)
}

// NewExt replaces or appends the extension to a file name.
func NewExt(name string, extension string) string {
	e := filepath.Ext(name)
	if e == "" {
		return name + extension
	}
	fn := strings.TrimSuffix(name, e)
	return fn + extension
}

// MakeThumb creates a thumb from an image that is size pixel in width and height.
func MakeThumb(file string, size int) {
	pfx := "_" + fmt.Sprintf("%v", size) + "x"
	cp := CopyFile(file, pfx)
	fmt.Println("Generating thumbnail x", size)
	src, err := imaging.Open(cp)
	checkErr(err)
	src = imaging.Resize(src, size, 0, imaging.Lanczos)
	src = imaging.CropAnchor(src, size, size, imaging.Center)
	// use the 3rd party CLI tool, pngquant to compress the PNG data
	src, err = pngquant.Compress(src, "4")
	err = imaging.Save(src, NewExt(cp, ".png"))
	checkErr(err)
	err = os.Remove(cp)
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

// CheckErr logs any errors
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

func dir(name string) {
	files, err := ioutil.ReadDir(name)
	checkErr(err)
	for _, file := range files {
		mime, err := mimetype.DetectFile(name + "/" + file.Name())
		checkErr(err)
		fmt.Println(file.Name(), humanize.Bytes(uint64(file.Size())), mime)
	}
}
