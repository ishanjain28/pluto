package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ishanjain28/pluto/pluto"

	humanize "github.com/dustin/go-humanize"
	flag "github.com/jessevdk/go-flags"
)

var Version string
var Build string

var options struct {
	Verbose      bool     `long:"verbose" description:"Enable Verbose Mode"`
	Connections  uint     `short:"n" long:"connections" default:"1" description:"Number of concurrent connections"`
	Name         string   `long:"name" description:"Path or Name of save file"`
	LoadFromFile string   `short:"f" long:"load-from-file" description:"Load URLs from a file"`
	Headers      []string `short:"H" long:"headers" description:"Headers to send with each request. Useful if a server requires some information in headers"`
	Version      bool     `short:"v" long:"version" description:"Print Pluto Version and exit"`
	urls         []string
}

func parseArgs() error {
	args, err := flag.ParseArgs(&options, os.Args)
	if err != nil {
		return fmt.Errorf("error parsing args: %v", err)
	}

	args = args[1:]

	options.urls = []string{}

	if options.LoadFromFile != "" {
		f, err := os.OpenFile(options.LoadFromFile, os.O_RDONLY, 0x444)
		if err != nil {
			return fmt.Errorf("error in opening file %s: %v", options.LoadFromFile, err)
		}
		defer f.Close()
		reader := bufio.NewReader(f)

		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("error in reading file: %v", err)
			}
			u := str[:len(str)-1]
			if u != "" {
				options.urls = append(options.urls, u)
			}
		}

		fmt.Printf("queued %d urls\n", len(options.urls))
	} else {
		for _, v := range args {
			if v != "" && v != "\n" {
				options.urls = append(options.urls, v)
			}
		}

	}
	if len(options.urls) == 0 {
		return fmt.Errorf("nothing to do. Please pass some url to fetch")
	}

	if options.Connections == 0 {
		return fmt.Errorf("connections should be > 0")
	}
	if len(options.urls) > 1 && options.Name != "" {
		return fmt.Errorf("it is not possible to specify 'name' with more than one url")
	}
	return nil
}
func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sig
		fmt.Printf("Interrupt Detected, Shutting Down.")

		if len(options.urls) > 0 {
			os.Exit(1)
		} else {
			cancel()
		}
	}()

	err := parseArgs()
	if err != nil {
		log.Fatalf("error parsing args: %v", err)
	}

	if options.Version {
		fmt.Println("Pluto - A Fast Multipart File Downloader")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build: %s\n", Build)
		return
	}

	for _, v := range options.urls {
		up, err := url.Parse(v)
		if err != nil {
			log.Printf("Invalid URL: %v", err)
			continue
		}
		p, err := pluto.New(up, options.Headers, options.Connections, options.Verbose)
		if err != nil {
			log.Fatalf("error creating pluto instance for url %s: %v\n", v, err)
		}

		go func() {
			if p.StatsChan == nil {
				return
			}

			for {
				select {
				case <-p.Finished:

					// Once download is finished, We don't need any data currently in channel.
					for range p.StatsChan {
					}

					break
				case v := <-p.StatsChan:
					os.Stdout.WriteString(fmt.Sprintf("\r%.2f%% - %s/%s - %s/s\r", float64(v.Downloaded)/float64(v.Size)*100, humanize.IBytes(v.Downloaded), humanize.IBytes(v.Size), humanize.IBytes(v.Speed)))
					os.Stdout.Sync()
				}
			}
		}()

		var fileName string
		if options.Name != "" {
			fileName = options.Name
		} else if p.MetaData.Name != "" {
			fileName = p.MetaData.Name
		} else {
			fileName = strings.Split(filepath.Base(up.String()), "?")[0]
		}
		fileName = strings.Replace(fileName, "/", "\\/", -1)
		writer, err := os.Create(fileName)
		if err != nil {
			log.Fatalf("unable to create file %s: %v\n", fileName, err)
		}
		defer writer.Close()

		if !p.MetaData.MultipartSupported && options.Connections > 1 {
			fmt.Printf("Downloading %s(%s) with 1 connection(Multipart downloads not supported)\n", fileName, humanize.Bytes(p.MetaData.Size))
		}

		result, err := p.Download(ctx, writer)
		if err != nil {
			log.Printf("error downloading url %s: %v", v, err)
		} else {
			s := humanize.IBytes(result.Size)
			htime := result.TimeTaken.String()

			as := humanize.IBytes(uint64(result.AvgSpeed))

			fmt.Printf("\nDownloaded %s in %s. Avg. Speed - %s/s\n", s, htime, as)
			fmt.Printf("File saved in %s\n", result.FileName)

		}

	}
}
