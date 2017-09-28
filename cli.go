package main

import (
	"bufio"
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
	flag "github.com/jessevdk/go-flags"

	"github.com/ishanjain28/pluto/pluto"
)

var Version string
var Build string

var options struct {
	Verbose bool `short:"v" long:"verbose" description:"Enable Verbose Mode"`

	Connections uint `short:"n" long:"connections" description:"Number of concurrent connections"`

	Name string `long:"name" description:"Path or Name of save file"`

	LoadFromFile string `short:"f" long:"load-from-file" description:"Load URLs from a file"`

	RetryCount uint `long:"retry-count" description:"Number of times a download should be retried in case of an error"`

	Version bool `long:"version" description:"Print Pluto Version and exit"`
}

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSTOP)

	go func() {
		<-sig
		fmt.Printf("Interrupt Detected, Shutting Down.")
		os.Exit(1)
	}()
	args, err := flag.ParseArgs(&options, os.Args)
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	if options.Version {
		fmt.Println("Pluto - A Fast Multipart File Downloader")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build: %s\n", Build)
		return
	}

	args = args[1:]

	urls := []string{}

	if options.LoadFromFile != "" {
		f, err := os.OpenFile(options.LoadFromFile, os.O_RDONLY, 0x444)
		if err != nil {
			log.Fatalf("error in opening file %s: %v\n", options.LoadFromFile, err)
		}
		defer f.Close()
		reader := bufio.NewReader(f)

		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("error in reading file: %v\n", err)
			}
			u := str[:len(str)-1]
			if u != "" {
				urls = append(urls, u)
			}
		}

		fmt.Printf("Queued %d urls\n", len(urls))
	} else {
		for _, v := range args {
			if v != "" && v != "\n" {
				urls = append(urls, v)
			}
		}

	}

	for i, v := range urls {
		up, err := url.Parse(v)
		if err != nil {
			log.Printf("Invalid URL: %v", err)
			continue
		}

		download(up, i)
	}
}

func download(up *url.URL, num int) {

	fname := strings.Split(filepath.Base(up.String()), "?")[0]

	meta, err := pluto.FetchMeta(up)
	if err != nil {
		log.Printf("error in fetching information about url: %v", err)
		return
	}

	if options.Name != "" && num == 0 {
		meta.Name = options.Name
	}

	if meta.Name == "" {
		meta.Name = fname
	}

	if meta.MultipartSupported || options.Connections == 0 {
		if options.Connections == 0 {
			options.Connections = 1
		}
		fmt.Printf("Downloading %s(%s) with %d connection\n", meta.Name, humanize.Bytes(meta.Size), options.Connections)
	} else {
		fmt.Printf("Downloading %s(%s) with 1 connection(Multipart downloads not supported)\n", meta.Name, humanize.Bytes(meta.Size))
	}

	saveFile, err := os.Create(strings.Replace(meta.Name, "/", "\\/", -1))
	if err != nil {
		log.Printf("error in creating file: %v", err)
		return
	}

	config := &pluto.Config{
		Meta:        meta,
		Connections: options.Connections,
		RetryCount:  options.RetryCount,
		Verbose:     options.Verbose,
		Writer:      saveFile,
		StatsChan:   make(chan *pluto.Stats),
	}

	startTime := time.Now()

	go func() {
		for v := range config.StatsChan {
			fmt.Printf("%s/%s - %s/s         \r", humanize.IBytes(v.Downloaded), humanize.IBytes(meta.Size), humanize.IBytes(v.Speed))
		}
	}()

	err = pluto.Download(config)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	timeTaken := time.Since(startTime)

	p, err := filepath.Abs(meta.Name)
	if err != nil {
		fmt.Printf("File saved in %s\n", meta.Name)
	}

	s := humanize.IBytes(meta.Size)
	htime := timeTaken.String()
	ts := timeTaken.Seconds()
	if ts == 0 {
		ts = 1
	}
	as := humanize.IBytes(uint64(float64(meta.Size) / float64(ts)))

	fmt.Printf("Downloaded %s in %s. Avg. Speed - %s/s\n", s, htime, as)
	fmt.Printf("File saved in %s\n", p)
}
