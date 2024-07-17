package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manifest struct {
	Pieces           []Piece `json:"pieces"`
	OriginalFileSize int64   `json:"originalFileSize"`
}

type Piece struct {
	URL        string `json:"url"`
	FileOffset int64  `json:"fileOffset"`
	FileSize   int64  `json:"fileSize"`
	HashValue  string `json:"hashValue"`
}

func fetchManifest(url string) (Manifest, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Manifest{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Manifest{}, err
	}
	defer resp.Body.Close()

	var manifest Manifest
	err = json.NewDecoder(resp.Body).Decode(&manifest)
	if err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func fetchPiece(url string, f *os.File, filesize int64, totalBytes *int64, startTime time.Time) ([2]string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return [2]string{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return [2]string{}, err
	}
	defer resp.Body.Close()

	chunksize := int64(1024 * 1024 * 5) // 5MiB chunk buff
	hsha1 := sha1.New()
	hsha256 := sha256.New()

	buffer := make([]byte, chunksize)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := f.Write(buffer[:n]); err != nil {
				return [2]string{}, err
			}
			hsha1.Write(buffer[:n])
			hsha256.Write(buffer[:n])
			*totalBytes += int64(n)
			elapsed := time.Since(startTime).Seconds()
			speed := float64(*totalBytes) / (1024 * 1024) / elapsed
			progress := int(100 * *totalBytes / filesize)
			filename := filepath.Base(f.Name())

			fmt.Printf("Downloading %s: % 6d%% (% 6.2fMiB/s)\r", filename, progress, speed)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return [2]string{}, err
		}
	}

	return [2]string{
		fmt.Sprintf("%x", hsha1.Sum(nil)),
		fmt.Sprintf("%x", hsha256.Sum(nil)),
	}, nil
}

func main() {
	output := flag.String("o", "", "Save pkg to PATH")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Usage: %s [options] URL\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	url := args[0]
	// if incorrect URL is provided, try to rewrite it into a correct one
	if strings.HasSuffix(url, "_sc.pkg") {
		url = url[:len(url)-7] + ".json"
	} else if strings.HasSuffix(url, "-DP.pkg") {
		url = url[:len(url)-7] + ".json"
	} else if strings.HasSuffix(url, "_0.pkg") {
		url = url[:len(url)-6] + ".json"
	}

	var filename string
	if *output != "" {
		filename = *output
	} else {
		filename = filepath.Base(url)[:len(filepath.Base(url))-4] + "pkg"
	}

	startTime := time.Now()
	manifest, err := fetchManifest(url)
	if err != nil {
		fmt.Printf("Error fetching manifest: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var totalBytes int64
	pieces := manifest.Pieces
	sort.Slice(pieces, func(i, j int) bool {
		return pieces[i].FileOffset < pieces[j].FileOffset
	})

	for _, piece := range pieces {
		offset, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			fmt.Printf("Error seeking file: %v\n", err)
			os.Exit(1)
		}

		if offset != piece.FileOffset {
			fmt.Printf("WARNING: inconsistent piece offset - expected %d, got %d\n", piece.FileOffset, offset)
		}

		hashes, err := fetchPiece(piece.URL, f, manifest.OriginalFileSize, &totalBytes, startTime)
		if err != nil {
			fmt.Printf("Error fetching piece: %v\n", err)
			os.Exit(1)
		}

		offset, err = f.Seek(0, io.SeekEnd)
		if err != nil {
			fmt.Printf("Error seeking file: %v\n", err)
			os.Exit(1)
		}

		if offset != piece.FileOffset+piece.FileSize {
			fmt.Printf("WARNING: inconsistent piece size - expected %d, got %d\n", piece.FileOffset+piece.FileSize, offset)
		}

		if !strings.EqualFold(piece.HashValue, hashes[0]) && !strings.EqualFold(piece.HashValue, hashes[1]) {
			fmt.Printf("WARNING: inconsistent piece hash - expected %s, got %s\n", piece.HashValue, hashes[0])
		}
	}

	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Printf("Error seeking file: %v\n", err)
		os.Exit(1)
	}

	if offset != manifest.OriginalFileSize {
		fmt.Printf("WARNING: inconsistent file size - expected %d, got %d\n", manifest.OriginalFileSize, offset)
	}

	name := filepath.Base(filename)
	size := float64(offset) / (1024 * 1024)
	speed := size / time.Since(startTime).Seconds()
	fmt.Printf("Completed %s: %.2fMiB (%.2fMiB/s)\n", name, size, speed)
}
