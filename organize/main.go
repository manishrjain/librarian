package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rwcarlsen/goexif/exif"

	_ "image/jpeg"
	_ "image/png"
)

var src = flag.String("src", "", "Choose directory to run this over")
var dst = flag.String("dst", "", "Choose root folder to move files to")
var dry = flag.Bool("dry", true, "Don't commit the changes."+
	" Only show what would be performed")
var cpy = flag.Bool("copy", false, "Copy instead of move. Does not affect duplicate deletion behavior.")
var deldups = flag.Bool("deletedups", false, "Delete duplicates present in source folder.")
var numroutines = flag.Int("numroutines", 2, "Number of routines to run.")

var dirs map[string]bool
var dirlocks DirLocks

// When running multiple goroutines to move files,
// we want to ensure that conflict resolution between
// files with the same generated name happens correctly.
// For that purpose, only one file move can happen
// per target directory at one time.
type DirLocks struct {
	locks   map[string]*sync.Mutex
	maplock sync.Mutex
}

func (d *DirLocks) Init() {
	d.locks = make(map[string]*sync.Mutex)
}

func (d *DirLocks) getLock(dir string) *sync.Mutex {
	d.maplock.Lock()
	if _, ok := d.locks[dir]; !ok {
		d.locks[dir] = new(sync.Mutex)
	}
	d.maplock.Unlock()

	m := d.locks[dir]
	return m
}

func (d *DirLocks) LockDir(dir string) {
	m := d.getLock(dir)
	m.Lock()
}

func (d *DirLocks) UnlockDir(dir string) {
	m := d.getLock(dir)
	m.Unlock()
}

func isVideo(ext string) bool {
	return ext == "mp4" || ext == "mov" || ext == "m4v"
}

type State struct {
	SrcPath string
	Sum     []byte
	Ext     string
	Ts      time.Time
}

func (s *State) Directory() string {
	dir := "Anarchs"
	if isVideo(s.Ext) {
		dir = "Videos"
	}
	if !s.Ts.IsZero() {
		dir = s.Ts.Format("2006.Jan")
	}
	return path.Join(*dst, dir)
}

func (s *State) PathWithoutExtension(full bool) string {
	name := ""
	if s.Ts.IsZero() {
		if full {
			name = fmt.Sprintf("%x", s.Sum)
		} else {
			name = fmt.Sprintf("%x", s.Sum[0:8])
		}
	} else {
		suffix := s.Sum[0:4]
		if full {
			suffix = s.Sum
		}
		name = fmt.Sprintf("%s_%x", s.Ts.Format("02_1504"), suffix)
	}
	folder := s.Directory()
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

func getType(f *os.File, path string) (string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}
	if _, t, err := image.Decode(f); err == nil {
		return t, nil
	}

	ext := filepath.Ext(path)
	ext = strings.ToLower(ext)
	if len(ext) > 1 {
		ext = ext[1:]
	}
	if isVideo(ext) {
		return ext, nil
	}

	return "", errors.New("Invalid file format")
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

func copyFile(s State) error {
	in, err := os.Open(s.SrcPath)
	if err != nil {
		return err
	}

	out, err := os.Create(s.ToPath())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// This is the function which does the heavy lifting of moving
// or deleting the duplicates. It's important that it gets a
// consistent read view of the final directory. For that purpose,
// we have a directory level mutex lock to ensure only one
// write operation happens at one time.
func moveFile(state State) error {
	dir := state.Directory()
	dirlocks.LockDir(dir)
	defer dirlocks.UnlockDir(dir)

	if err := dirExists(dir); err != nil {
		return err
	}

	pattern := state.PathWithoutExtension(false) + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		if *cpy {
			fmt.Printf("Copying %s to %s\n", state.SrcPath, state.ToPath())
		} else {
			fmt.Printf("Moving %s to %s\n", state.SrcPath, state.ToPath())
		}
		if *dry {
			return nil
		}

		if *cpy {
			return copyFile(state)
		} else {
			return os.Rename(state.SrcPath, state.ToPath())
		}
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
			fmt.Printf("Already exists: %s\n", dup)
			if *dry || !*deldups {
				return nil
			}
			fmt.Printf("DELETING %s\n", state.SrcPath)
			return os.Remove(state.SrcPath)
		}
	}

	// Doesn't match with any of the existing files.
	// Let's move this image
	return nil
}

func handleFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var state State
	state.SrcPath = path
	if state.Ext, err = getType(f, path); err != nil {
		fmt.Printf("%s: Not an image file. Moving on...\n", path)
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

var lch chan string

func routine(wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range lch {
		if err := handleFile(path); err != nil {
			panic(err)
		}
	}
}

func shuffle(a []string) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	if *src == "" || *dst == "" {
		flag.Usage()
		return
	}
	if *dry {
		fmt.Println("DRY mode. No changes would be committed.")
	}

	dirs = make(map[string]bool)
	dirlocks.Init()

	lch = make(chan string)
	wg := new(sync.WaitGroup)

	fmt.Printf("Using %d routines\n", *numroutines)
	for i := 0; i < *numroutines; i++ {
		wg.Add(1)
		go routine(wg)
	}

	var total int64
	var l []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		total += info.Size()
		l = append(l, path)
		return nil
	}

	if err := filepath.Walk(*src, walkFn); err != nil {
		panic(err)
	}
	fmt.Printf("Found %d files of size: %.2f GB\n", len(l), float64(total)/(1024*1024*1024))
	// Shuffle so our dir locks can avoid contention due to time locality of
	// images, present next to each other in the source folder.
	shuffle(l)

	for _, path := range l {
		lch <- path
	}

	close(lch)
	fmt.Println("Closed channel. Waiting...")
	wg.Wait()
	fmt.Println("Done waiting.")
}
