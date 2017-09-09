package main

import (
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

	"github.com/ishanjain28/pluto/pluto"
)

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Printf("Interrupt Detected, Shutting Down.")
		os.Exit(0)
	}()

	parts := flag.Int("part", 32, "Number of Download parts")
	verbose := flag.Bool("verbose", false, "Enable Verbose Mode")
	name := flag.String("name", "pluto_download", "Path or Name of save File")

	flag.Parse()

	urls := []string{}

	for i, v := range os.Args {
		if i == 0 || strings.Contains(v, "-name=") || strings.Contains(v, "-part=") || strings.Contains(v, "-verbose") {
			continue
		}

		urls = append(urls, v)
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

func download(u, filename string, parts int, verbose bool) {
	a := time.Now()

	// defer func() { select {} }()

	up, err := url.Parse(u)
	if err != nil {
		log.Println("Invalid URL")
		return
	}

	fname := strings.Split(filepath.Base(up.String()), "?")[0]
	fmt.Printf("Starting Download with %d parts\n", parts)

	fmt.Printf("Downloading %s\n", up.String())

	saveFile, err := os.Create(filename)
	if err != nil {
		log.Fatalln("error in creating save file: %v", err)
	}

	meta, err := pluto.FetchMeta(up)
	if err != nil {
		log.Fatalln("error in fetching information about url: %v", err)
	}

	config := &pluto.Config{
		Meta:       meta,
		Parts:      parts,
		RetryCount: 10,
		Verbose:    verbose,
		Writer:     saveFile,
	}

	err = pluto.Download(config)
	if err != nil {
		log.Println(err)
		return
	}

	file, err := os.Create(fname)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer file.Close()

	fmt.Printf("Downloaded %s in %s\n", up.String(), time.Since(a))
	t, err := io.Copy(file, saveFile)
	if err != nil {
		log.Println(err.Error())
	}

	fmt.Println("bytes saved", t)
}
