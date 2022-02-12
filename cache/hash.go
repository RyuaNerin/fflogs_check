package cache

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
)

func cleanUpWithHash(dir string, dirForHash ...string) {
	newHash := hashDir(dirForHash...)
	var oldHash uint32

	b := make([]byte, 4)

	hashFile := filepath.Join(dir, "hash")
	fs, err := os.OpenFile(hashFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0400)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		oldHash = newHash + 1
	} else {
		defer fs.Close()

		_, err = fs.Read(b)
		if err != nil && err != io.EOF {
			panic(err)
		}

		oldHash = binary.BigEndian.Uint32(b)
	}

	if newHash != oldHash {
		os.RemoveAll(dir)

		os.MkdirAll(dir, 0700)

		binary.BigEndian.PutUint32(b, newHash)
		_, err = fs.Write(b)
		if err != nil {
			panic(err)
		}
	}
}

func hashDir(dir ...string) uint32 {
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

	walk = func(dir string) {
		fiList, err := os.ReadDir(dir)
		if err != nil {
			panic(err)
		}

		for _, fi := range fiList {
			path := filepath.Join(dir, fi.Name())
			fmt.Fprint(h, path)

			if fi.IsDir() {
				walk(path)
			} else {
				read(path)
			}
		}
	}

	for _, d := range dir {
		walk(d)
	}

	return h.Sum32()
}
