package cache

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"

	"github.com/getsentry/sentry-go"
)

func cleanUpWithHash(dir string, hashPath ...string) {
	b := make([]byte, 4)

	hashFile := filepath.Join(dir, "hash")

	newHash := hashDir(hashPath...)
	oldHash := func() uint32 {
		binary.BigEndian.PutUint32(b, newHash+1)

		fs, err := os.Open(hashFile)
		if err != nil {
			return newHash + 1
		}
		defer fs.Close()

		_, err = fs.Read(b)
		if err != nil && err != io.EOF {
			sentry.CaptureException(err)
			return newHash + 1
		}

		return binary.BigEndian.Uint32(b)
	}()

	if newHash != oldHash {
		os.RemoveAll(dir)

		os.MkdirAll(dir, 0700)

		binary.BigEndian.PutUint32(b, newHash)
		os.WriteFile(hashFile, b, 0600)
	}
}

func hashDir(pathList ...string) uint32 {
	h := fnv.New32a()

	read := func(path string) {
		fs, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(h, fs)
		if err != nil && err != io.EOF {
			panic(err)
		}
	}

	var walk func(dir string)
	walk = func(dirPath string) {
		fiList, err := os.ReadDir(dirPath)
		if err != nil {
			panic(err)
		}

		for _, fi := range fiList {
			path := filepath.Join(dirPath, fi.Name())
			fmt.Fprint(h, path)

			if fi.IsDir() {
				walk(path)
			} else {
				read(path)
			}
		}
	}

	for _, path := range pathList {
		stat, err := os.Stat(path)
		if err != nil {
			panic(err)
		}
		if stat.IsDir() {
			walk(path)
		} else {
			read(path)
		}
	}

	return h.Sum32()
}
