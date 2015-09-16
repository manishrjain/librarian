package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var dir = flag.String("dir", "", "Choose directory to dedup video files.")
var deletedups = flag.Bool("deletedups", false, "Delete duplicate videos.")
var percent = flag.Float64("percent", 95.0, "Video matching threshold ratio.")

type Video struct {
	Path      string
	Size      int64
	Checksums [][32]byte
}

func processVideo(video *Video) error {
	f, err := os.Open(video.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := make([]byte, 32*1024)
	for {
		n, err := f.Read(buffer)
		if n == 0 && err == io.EOF {
			break

		} else if err != nil {
			return err
		}

		h := sha256.Sum256(buffer[0:n])
		video.Checksums = append(video.Checksums, h)
	}
	return nil
}

func ratioMatch(v1, v2 Video) float64 {
	if len(v1.Checksums) != len(v2.Checksums) {
		return 0.0
	}
	if len(v1.Checksums) == 0 {
		return 0.0
	}

	matches := 0
	for idx, h1 := range v1.Checksums {
		h2 := v2.Checksums[idx]
		if bytes.Equal(h1[:], h2[:]) {
			matches += 1
		}
	}
	total := len(v1.Checksums)
	ratio := float64(matches) * 100.0 / float64(total)
	return ratio
}

func main() {
	flag.Parse()

	if *dir == "" {
		flag.Usage()
		return
	}

	var videos []*Video
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		v := new(Video)
		v.Path = path
		v.Size = info.Size()
		videos = append(videos, v)
		return nil
	}
	filepath.Walk(*dir, walkFn)

	matches := 0
	var potential float64
	for i := range videos {
		for j := i + 1; j < len(videos); j++ {
			v1 := videos[i]
			v2 := videos[j]
			if v1.Size != v2.Size {
				continue
			}
			// Sizes are same. Checksum now.
			if len(v1.Checksums) == 0 {
				if err := processVideo(v1); err != nil {
					panic(err)
				}
			}
			if len(v2.Checksums) == 0 {
				if err := processVideo(v2); err != nil {
					panic(err)
				}
			}
			ratio := ratioMatch(*v1, *v2)
			if ratio < *percent {
				fmt.Printf("Equal checksum, but NO Match: %.2f for [%q] [%q]\n",
					ratio, v1.Path, v2.Path)

			} else {
				matches += 1
				potential += float64(v1.Size)
				fmt.Printf("Match: %.2f for [%q] [%q]\n", ratio, v1.Path, v2.Path)
				if *deletedups {
					fmt.Printf("DELETING: [%q]\n", v1.Path)
					if err := os.Remove(v1.Path); err != nil {
						panic(err)
					}
				}
				break // No need to check v1 against any other videos.
			}
		}
	}

	fmt.Printf("Matches found: %d\n", matches)
	fmt.Printf("Potential Save: %.2f MB\n", potential/(1024*1024))
	if *deletedups {
		fmt.Printf("DELETED files: %d\n", matches)
	}
}
