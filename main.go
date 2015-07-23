package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"

	_ "image/jpeg"
	_ "image/png"
)

var src = flag.String("src", "", "Choose directory to run this over")
var dst = flag.String("dst", "", "Choose root folder to move files to")
var dry = flag.Bool("dry", true, "Don't commit the changes."+
	" Only show what would be performed")

var dirs map[string]bool

func getType(f *os.File) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}
	_, t, err := image.Decode(f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return "", err
	}
	return t, nil
}

func getSum(f *os.File) (csum []byte, rerr error) {
	if _, err := f.Seek(0, 0); err != nil {
		return csum, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return csum, err
	}
	csum = h.Sum(nil)
	return csum, nil
}

func dirExists(dir string) error {
	if exists := dirs[dir]; exists {
		return nil
	}

	_, err := os.Stat(dir)
	if err == nil {
		dirs[dir] = true
		return nil
	}
	if os.IsNotExist(err) {
		fmt.Printf("Creating directory: %v\n", dir)
		if merr := os.MkdirAll(dir, 0755); merr != nil {
			return merr
		}
	}
	dirs[dir] = true
	return nil
}

func getFinalPath(f *os.File, ftype string, csum []byte) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}

	dfolder := "Anarchs"
	fname := ""
	x, err := exif.Decode(f)
	if err == nil {
		dt, ierr := x.DateTime()
		if ierr == nil {
			dfolder = dt.Format("2006Jan")
			fname = fmt.Sprintf("%s_%x", dt.Format("02_1504"), csum[0:4])
		}
	}
	dfolder = path.Join(*dst, dfolder)
	if err := dirExists(dfolder); err != nil {
		fmt.Printf("Error checking/creating directory: %v\n", err)
		return "", err
	}

	if fname == "" {
		fname = fmt.Sprintf("%x", csum[0:8])
	}
	fname = fname + "." + ftype
	return path.Join(dfolder, fname), nil
}

func moveFile(src, dst string) {
}

func handleFile(path string) error {
	fmt.Println("Considering " + path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := getType(f)
	if err != nil {
		fmt.Println("Not an image file. Moving on...")
		return nil
	}

	csum, err := getSum(f)
	if err != nil {
		return err
	}

	fpath, err := getFinalPath(f, t, csum)
	if err != nil {
		return err
	}

	// We already have the folder as the YYYYMMM,
	// so no need to have that part in the file name.
	fmt.Println("Final path: " + fpath)
	return nil
}

func walkFn(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		fmt.Printf("Directory: %v\n", path)
		return nil
	}

	return handleFile(path)
}

func main() {
	flag.Parse()
	if *src == "" || *dst == "" {
		flag.Usage()
		return
	}
	dirs = make(map[string]bool)

	if err := filepath.Walk(*src, walkFn); err != nil {
		panic(err)
	}
}
