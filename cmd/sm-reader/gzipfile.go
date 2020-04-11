package main

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

// gzipFile gzips the file with the given name
func gzipFile(filename string) {
	src, err := os.Open(filename)
	if err != nil {
		log.Fatal("Could not open file for reading")
	}

	defer src.Close()

	// Open file for writing.
	dest, err := os.Create(filename + ".gz")
	if err != nil {
		log.Fatal("Could not open file for writing")
	}

	defer dest.Close()

	// Create a Reader and use ReadAll to get all the bytes from the file.
	reader := bufio.NewReader(src)

	// Write compressed data.
	writer, _ := gzip.NewWriterLevel(dest, 9)

	if _, err := io.Copy(writer, reader); err != nil {
		log.Fatal("Could not gzip the contents of the file")
	}

	writer.Close()
}
