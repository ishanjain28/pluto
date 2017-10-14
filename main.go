package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	humanize "github.com/dustin/go-humanize"
	flag "github.com/jessevdk/go-flags"

	"github.com/ishanjain28/pluto/pluto"
)

var Version string
var Build string

var options struct {
	Verbose bool `long:"verbose" description:"Enable Verbose Mode"`

	Connections uint `short:"n" long:"connections" default:"1" description:"Number of concurrent connections"`

	Name string `long:"name" description:"Path or Name of save file"`

	LoadFromFile string `short:"f" long:"load-from-file" description:"Load URLs from a file"`

	Headers []string `short:"H" long:"Headers" description:"Headers to send with each request. Useful if a server requires some information in headers"`

	Version bool `short:"v" long:"version" description:"Print Pluto Version and exit"`
}

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

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
	if len(urls) == 0 {
		log.Fatalf("nothing to do. Please pass some url to fetch")
	}

	if options.Connections == 0 {
		log.Fatalf("Connections should be > 0")
	}
	if len(urls) > 1 && options.Name != "" {
		log.Fatalf("it is not possible to specify 'name' with more than one url")
	}

	for _, v := range urls {
		up, err := url.Parse(v)
		if err != nil {
			log.Printf("Invalid URL: %v", err)
			continue
		}

		p, err := pluto.New(up, options.Headers, options.Name, options.Connections, options.Verbose)
		if err != nil {
			log.Printf("error creating pluto instance for url %s: %v", v, err)
		}
		go func() {
			if p.StatsChan == nil {
				return
			}

			for {
				select {
				case <-p.Finished:
					break
				case v := <-p.StatsChan:
					os.Stdout.WriteString(fmt.Sprintf("%.2f%% - %s/%s - %s/s	   	      \r", float64(v.Downloaded)/float64(v.Size)*100, humanize.IBytes(v.Downloaded), humanize.IBytes(v.Size), humanize.IBytes(v.Speed)))
					os.Stdout.Sync()
				}

			}

		}()
		result, err := p.Download()
		if err != nil {
			log.Printf("error downloading url %s: %v", v, err)
		} else {
			s := humanize.IBytes(result.Size)
			htime := result.TimeTaken.String()

			as := humanize.IBytes(uint64(result.AvgSpeed))

			fmt.Printf("Downloaded %s in %s. Avg. Speed - %s/s\n", s, htime, as)
			fmt.Printf("File saved in %s\n", result.FileName)

		}

	}
}
