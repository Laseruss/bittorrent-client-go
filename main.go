package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
)

func main() {
	filename := ""
	outname := ""
	flag.StringVar(&filename, "path", "", "path to the torrent file")
	flag.StringVar(&outname, "out", "", "name of the created file")
	flag.Parse()

	if filename == "" || outname == "" {
		fmt.Fprintf(os.Stderr, "need a path and an out name\n")
		os.Exit(1)
	}

	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open file: %s\n", err)
		os.Exit(1)
	}

	t, err := newTorrent(f)
	if err != nil {
		fmt.Printf("could not parse file into torrent: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully parsed the torrent file")

	file, err := Download(t)
	if err != nil {
		fmt.Println("could not download the file", err)
		os.Exit(1)
	}

	os.WriteFile(outname, file, fs.ModePerm)
}
