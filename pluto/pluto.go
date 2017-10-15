// Package pluto provides a way to download files at high speeds by using http ranged requests.
package pluto

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	humanize "github.com/dustin/go-humanize"

	"time"
)

var (

	// ErrOverflow is when server sends more data than what was requested
	ErrOverflow = "error: Server sent extra bytes"
)

// Stats is returned in a channel by Download function every 250ms and contains details like Current download speed in bytes/sec and amount of data Downloaded
type Stats struct {
	Downloaded uint64
	Speed      uint64
	Size       uint64
}

type fileMetaData struct {
	url                *url.URL
	Size               uint64
	Name               string
	MultipartSupported bool
}

// Pluto contains all the details that Download needs.
// Connections is the number of connections to use to download a file
// Verbose is to enable verbose mode.
// Writer is the place where downloaded data is written.
// Headers is any header that you may need to send to download the file.
// StatsChan is a channel to which Stats are sent, It can be nil or a channel that can hold data of type *()
type Pluto struct {
	StatsChan   chan *Stats
	Finished    chan struct{}
	connections uint
	verbose     bool
	headers     []string
	downloaded  uint64
	MetaData    fileMetaData
	startTime   time.Time
	workers     []*worker
}

//Result is the download results
type Result struct {
	FileName  string
	Size      uint64
	AvgSpeed  float64
	TimeTaken time.Duration
}

//New returns a pluto instance
func New(up *url.URL, headers []string, connections uint, verbose bool) (*Pluto, error) {

	p := &Pluto{
		connections: connections,
		headers:     headers,
		verbose:     verbose,
		StatsChan:   make(chan *Stats),
		Finished:    make(chan struct{}),
	}

	err := p.fetchMeta(up, headers)
	if err != nil {
		return nil, err
	}

	if !p.MetaData.MultipartSupported {
		p.connections = 1
		fmt.Printf("Downloading %s(%s) with %d connection\n", p.MetaData.Name, humanize.Bytes(p.MetaData.Size), p.connections)
	} else {
		p.connections = 1
	}

	return p, nil

}

// Download takes Config struct
// then downloads the file by dividing it into given number of parts and downloading all parts concurrently.
// If any error occurs in the downloading stage of any part, It'll check if the the program can recover from error by retrying download
// And if an error occurs which the program can not recover from, it'll return that error
func (p *Pluto) Download(ctx context.Context, w io.WriterAt) (*Result, error) {
	p.startTime = time.Now()
	// Limit number of CPUs it can use
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)

	perPartLimit := p.MetaData.Size / uint64(p.connections)
	difference := p.MetaData.Size % uint64(p.connections)

	p.workers = make([]*worker, p.connections)

	for i := uint(0); i < p.connections; i++ {
		begin := perPartLimit * uint64(i)
		end := perPartLimit * (uint64(i) + 1)

		if i == p.connections-1 {
			end += difference
		}

		p.workers[i] = &worker{
			begin:   begin,
			end:     end,
			url:     p.MetaData.url,
			writer:  w,
			headers: p.headers,
			verbose: p.verbose,
			ctx:     ctx,
		}
	}

	err := p.startDownload()
	if err != nil {
		return nil, err
	}

	tt := time.Since(p.startTime)
	filename, err := filepath.Abs(p.MetaData.Name)
	if err != nil {
		log.Printf("unable to get absolute path for %s: %v", p.MetaData.Name, err)
		filename = p.MetaData.Name
	}

	r := &Result{
		TimeTaken: tt,
		FileName:  filename,
		Size:      p.MetaData.Size,
		AvgSpeed:  float64(p.MetaData.Size) / float64(tt.Seconds()),
	}

	close(p.Finished)
	return r, nil
}

func (p *Pluto) startDownload() error {

	var wg sync.WaitGroup
	wg.Add(len(p.workers))
	var err error

	errdl := make(chan error, 1)
	errcopy := make(chan error, 1)

	var downloaded uint64

	// Stats system, It writes stats to the stats channel
	go func() {

		var oldSpeed uint64
		counter := 0
		for {

			dled := atomic.LoadUint64(&downloaded)
			speed := dled - p.downloaded

			if speed == 0 && counter < 4 {
				speed = oldSpeed
				counter++
			} else {
				counter = 0
			}

			p.StatsChan <- &Stats{
				Downloaded: p.downloaded,
				Speed:      speed * 2,
				Size:       p.MetaData.Size,
			}

			p.downloaded = dled
			oldSpeed = speed
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for _, w := range p.workers {
		// This loop keeps trying to download a file if a recoverable error occurs
		go func(w *worker, wgroup *sync.WaitGroup, dl *uint64, cerr, dlerr chan error) {

			defer func() {

				wgroup.Done()
				cerr <- nil
				dlerr <- nil
			}()

			for {
				downloadPart, err := w.download()
				if err != nil {
					if err.Error() == "status code: 400" || err.Error() == "status code: 500" || err.Error() == ErrOverflow {
						cerr <- err
						return
					}

					if p.verbose {
						log.Println(err)
					}
					continue
				}

				d, err := w.copyAt(downloadPart, &downloaded)
				if err != nil {
					cerr <- fmt.Errorf("error copying data at offset %d: %v", w.begin, err)
				}

				if p.verbose {
					fmt.Printf("Copied %d bytes\n", d)
				}

				downloadPart.Close()
				break
			}

		}(w, &wg, &downloaded, errcopy, errdl)
	}

	err = <-errcopy
	if err != nil {
		return err
	}

	err = <-errdl
	if err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func (p *Pluto) fetchMeta(u *url.URL, headers []string) error {

	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return fmt.Errorf("error in creating HEAD request: %v", err)
	}

	for _, v := range headers {
		vsp := strings.Index(v, ":")

		key := v[:vsp]
		value := v[vsp:]

		req.Header.Set(key, value)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error in sending HEAD request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("status code is %d", resp.StatusCode)
	}

	size := resp.ContentLength
	if size == 0 {
		return fmt.Errorf("Incompatible URL, file size is 0")
	}

	msupported := false

	if resp.Header.Get("Accept-Range") != "" || resp.Header.Get("Accept-Ranges") != "" {
		msupported = true
	}

	resp, err = http.Get(u.String())
	if err != nil {
		return fmt.Errorf("error in sending GET request: %v", err)
	}

	name := ""

	dispositionHeader := resp.Header.Get("Content-Disposition")

	if dispositionHeader != "" {
		cDispose := strings.Split(dispositionHeader, "filename=")

		if len(cDispose) > 0 {
			cdfilename := cDispose[1]
			cdfilename = cdfilename[1:]
			cdfilename = cdfilename[:len(cdfilename)-1]
			name = cdfilename
		}
	}

	resp.Body.Close()
	p.MetaData = fileMetaData{
		Size:               uint64(size),
		Name:               name,
		url:                u,
		MultipartSupported: msupported,
	}
	return nil
}
