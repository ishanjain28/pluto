package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/ishanjain28/pluto/pluto"
)

var Version string
var Build string

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Printf("Interrupt Detected, Shutting Down.")
		os.Exit(0)
	}()

	parts := flag.Uint("part", 32, "Number of Download parts")
	verbose := flag.Bool("verbose", false, "Enable Verbose Mode")
	name := flag.String("name", "", "Path or Name of save File")
	loadFromFile := flag.String("load-from-file", "", "Load URLs from a file")
	version := flag.Bool("version", false, "Pluto Version")
	flag.Parse()

	if *version {
		fmt.Println("Pluto - A Fast Multipart File Downloader")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build: %s\n", Build)
		return
	}

	urls := []string{}

	if *loadFromFile == "" {
		for i, v := range os.Args {
			if i == 0 || strings.Contains(v, "-load-from-file") || strings.Contains(v, "-name=") || strings.Contains(v, "-part=") || strings.Contains(v, "-verbose") {
				continue
			}

			urls = append(urls, v)
		}
	} else {
		f, err := os.OpenFile(*loadFromFile, os.O_RDONLY, 0x444)
		if err != nil {
			log.Fatalf("error in opening file %s: %v\n", *loadFromFile, err)
		}
		defer f.Close()
		reader := bufio.NewReader(f)

		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("error in reading file: %v", err)
			}

			urls = append(urls, str[:len(str)-1])
		}
	}

	if len(urls) == 0 {
		u := ""
		fmt.Printf("URL: ")
		fmt.Scanf("%s\n", &u)
		if u == "" {
			log.Fatalln("No URL Provided")
		}
		urls = append(urls, u)
	}
	for _, v := range urls {
		download(v, *name, *parts, *verbose)
	}
}

func download(u, filename string, parts uint, verbose bool) {

	// This variable is used to check if an error occurred anywhere in program
	// If an error occurs, Then it'll not exit.
	// And if no error occurs, Then it'll exit after 10 seconds
	var errored bool
	var dlFinished bool

	defer func() {
		if errored {
			select {}
		} else {
			time.Sleep(5 * time.Second)
		}
	}()

	up, err := url.Parse(u)
	if err != nil {
		errored = true
		log.Println("Invalid URL")
		return
	}

	fname := strings.Split(filepath.Base(up.String()), "?")[0]
	fmt.Printf("Starting Download with %d connections\n", parts)

	fmt.Printf("\nDownloading %s\n", up.String())

	meta, err := pluto.FetchMeta(up)
	if err != nil {
		errored = true
		log.Printf("error in fetching information about url: %v", err)
		return
	}

	if filename != "" {
		meta.Name = filename
	}

	if meta.Name == "" {
		meta.Name = fname
	}

	fmt.Println(meta.Name)
	saveFile, err := os.Create(meta.Name)
	if err != nil {
		errored = true
		log.Printf("error in creating save file: %v", err)
		return
	}

	config := &pluto.Config{
		Meta:       meta,
		Parts:      parts,
		RetryCount: 10,
		Verbose:    verbose,
		Writer:     saveFile,
	}

	startTime := time.Now()
	go func() {
		for {
			time.Sleep(1 * time.Second)
			elapsed := time.Since(startTime)

			avgSpeed := meta.Size / uint64(elapsed.Seconds())

			time.Sleep(100 * time.Millisecond)

			if dlFinished {
				break
			}
			fmt.Printf("Average Speed: %s/s, Elapsed Time: %s\r", humanize.IBytes(avgSpeed), elapsed.String())
		}
	}()

	err = pluto.Download(config)
	if err != nil {
		errored = true
		log.Printf("%v", err)
		return
	}
	timeTaken := time.Since(startTime)
	dlFinished = true
	fmt.Printf("Downloaded complete in %s. Avg. Speed - %s/s\n", timeTaken, humanize.IBytes(meta.Size/uint64(timeTaken.Seconds())))

	p, err := filepath.Abs(meta.Name)
	if err != nil {
		fmt.Printf("File saved in %s\n", meta.Name)
	}
	fmt.Printf("File saved in %s\n", p)
}
