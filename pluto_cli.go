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
	flag.Parse()

	urls := []string{}

	for i, v := range os.Args {
		if i == 0 || strings.Contains(v, "-part=") || strings.Contains(v, "-verbose") {
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
		download(v, *parts, *verbose)
	}
}

func download(u string, parts int, verbose bool) {
	a := time.Now()

	defer func() { select {} }()

	up, err := url.Parse(u)
	if err != nil {
		log.Println("Invalid URL")
		return
	}

	fname := strings.Split(filepath.Base(up.String()), "?")[0]
	fmt.Printf("Starting Download with %d parts\n", parts)

	fmt.Printf("Downloading %s\n", up.String())

	f, err := pluto.Download(up, parts, verbose)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	file, err := os.Create(fname)
	if err != nil {
		log.Println(err.Error())
		return
	}

	defer file.Close()
	defer os.Remove(f.Name())

	fmt.Printf("Downloaded %s in %s\n", up.String(), time.Since(a))
	_, err = io.Copy(file, f)
	if err != nil {
		log.Println(err.Error())
	}
}
