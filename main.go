package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/rwcarlsen/goexif/exif"

	_ "image/jpeg"
	_ "image/png"
)

var src = flag.String("src", "", "Choose directory to run this over")
var dst = flag.String("dst", "", "Choose root folder to move files to")
var dry = flag.Bool("dry", true, "Don't commit the changes."+
	" Only show what would be performed")

var dirs map[string]bool

type State struct {
	SrcPath string
	Sum     []byte
	Ext     string
	Ts      time.Time
}

func (s *State) PathWithoutExtension(full bool) string {
	dir := "Anarchs"
	name := ""
	if s.Ts.IsZero() {
		if full {
			name = fmt.Sprintf("%x", s.Sum)
		} else {
			name = fmt.Sprintf("%x", s.Sum[0:8])
		}
	} else {
		dir = s.Ts.Format("2006Jan")
		suffix := s.Sum[0:4]
		if full {
			suffix = s.Sum
		}
		name = fmt.Sprintf("%s_%x", s.Ts.Format("02_1504"), suffix)
	}
	folder := path.Join(*dst, dir)
	return path.Join(folder, name)
}

func (s *State) ToPath() string {
	path := s.PathWithoutExtension(false)
	return path + "." + s.Ext
}

func (s *State) LongPath() string {
	path := s.PathWithoutExtension(true)
	return path + "." + s.Ext
}

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

func getTimestamp(f *os.File) (rts time.Time, rerr error) {
	if _, err := f.Seek(0, 0); err != nil {
		return rts, err
	}

	x, err := exif.Decode(f)
	if err == nil {
		if ts, ierr := x.DateTime(); ierr == nil {
			return ts, nil
		}
	}
	return rts, errors.New("Unable to find ts")
}

func moveFile(state State) error {
	pattern := state.PathWithoutExtension(false) + "*"
	fmt.Printf("Pattern: %v\n", pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	fmt.Println(matches)
	if len(matches) == 0 {
		fmt.Printf("Moving %s to %s\n", state.SrcPath, state.ToPath())
		if *dry {
			return nil
		}
		return os.Rename(state.SrcPath, state.ToPath())
	}

	for _, dup := range matches {
		f, err := os.Open(dup)
		if err != nil {
			return err
		}
		dupsum, err := getSum(f)
		if err != nil {
			return err
		}
		if bytes.Equal(state.Sum, dupsum) {
			// src is a duplicate of a file which already is copied to destination.
			fmt.Printf("Already exists: %s. Deleting %s", dup, state.SrcPath)
			if *dry {
				return nil
			}
			return os.Remove(state.SrcPath)
		}
	}

	// Doesn't match with any of the existing files.
	// Let's move this image
	return nil
}

func handleFile(path string) error {
	fmt.Println("Considering " + path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var state State
	state.SrcPath = path
	if state.Ext, err = getType(f); err != nil {
		fmt.Println("Not an image file. Moving on...")
		return nil
	}

	if state.Sum, err = getSum(f); err != nil {
		return err
	}

	if state.Ts, err = getTimestamp(f); err != nil {
		state.Ts = time.Time{}
	}

	// We already have the folder as the YYYYMMM,
	// so no need to have that part in the file name.
	return moveFile(state)
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
