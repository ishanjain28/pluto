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

	parts := flag.Int("part", 16, "Number of Download parts")

	flag.Parse()
	urls := []string{}
	u := ""

	if len(os.Args) <= 1 {
		fmt.Printf("URL: ")
		fmt.Scanf("%s\n", &u)
		if u == "" {
			log.Fatalln("No URL Provided")
		}

		download(u, *parts)
	} else {

		if *parts == 0 {
			urls = os.Args[1:]
		} else {
			urls = os.Args[2:]
		}

		for _, v := range urls {
			download(v, *parts)
		}
	}

}

func download(u string, parts int) {
	a := time.Now()

	up, err := url.Parse(u)
	if err != nil {
		log.Fatalln("Invalid URL")
	}

	fname := filepath.Base(up.String())

	fmt.Printf("Downloading %s\n", up.String())

	f, err := pluto.Download(up, parts)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	file, err := os.Create(fname)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer file.Close()
	defer os.Remove(f.Name())

	fmt.Printf("Downloaded %s in %s\n", up.String(), time.Since(a))
	_, err = io.Copy(file, f)
	if err != nil {
		log.Fatalln(err.Error())
	}
}
