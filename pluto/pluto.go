// Package pluto provides a way to download files at high speeds by using http ranged requests.
package pluto

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

type worker struct {
	begin uint64
	end   uint64
	url   *url.URL
}

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
	metaData    fileMetaData
	startTime   time.Time
	writer      io.WriterAt
}

//Result is the download results
type Result struct {
	FileName  string
	Size      uint64
	AvgSpeed  float64
	TimeTaken time.Duration
}

//New returns a pluto instance
func New(up *url.URL, headers []string, name string, connections uint, verbose bool) (*Pluto, error) {

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

	if name != "" {
		p.metaData.Name = name
	} else if p.metaData.Name == "" {
		p.metaData.Name = strings.Split(filepath.Base(up.String()), "?")[0]
	}

	if p.metaData.MultipartSupported {
		fmt.Printf("Downloading %s(%s) with %d connection\n", p.metaData.Name, humanize.Bytes(p.metaData.Size), p.connections)
	} else {
		fmt.Printf("Downloading %s(%s) with 1 connection(Multipart downloads not supported)\n", p.metaData.Name, humanize.Bytes(p.metaData.Size))
		p.connections = 1
	}

	p.writer, err = os.Create(strings.Replace(p.metaData.Name, "/", "\\/", -1))
	if err != nil {
		return nil, fmt.Errorf("error creating file %s: %v", p.metaData.Name, err)
	}

	return p, nil

}

// Download takes Config struct
// then downloads the file by dividing it into given number of parts and downloading all parts concurrently.
// If any error occurs in the downloading stage of any part, It'll check if the the program can recover from error by retrying download
// And if an error occurs which the program can not recover from, it'll return that error
func (p *Pluto) Download() (*Result, error) {
	p.startTime = time.Now()
	// Limit number of CPUs it can use
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)

	perPartLimit := p.metaData.Size / uint64(p.connections)
	difference := p.metaData.Size % uint64(p.connections)

	workers := make([]*worker, p.connections)

	for i := uint(0); i < p.connections; i++ {
		begin := perPartLimit * uint64(i)
		end := perPartLimit * (uint64(i) + 1)

		if i == p.connections-1 {
			end += difference
		}

		workers[i] = &worker{
			begin: begin,
			end:   end,
			url:   p.metaData.url,
		}
	}

	err := p.startDownload(workers)
	if err != nil {
		return nil, err
	}

	tt := time.Since(p.startTime)
	filename, err := filepath.Abs(p.metaData.Name)
	if err != nil {
		log.Printf("unable to get absolute path for %s: %v", p.metaData.Name, err)
		filename = p.metaData.Name
	}

	r := &Result{
		TimeTaken: tt,
		FileName:  filename,
		Size:      p.metaData.Size,
		AvgSpeed:  float64(p.metaData.Size) / float64(tt.Seconds()),
	}

	close(p.Finished)
	return r, nil
}

func (p *Pluto) startDownload(w []*worker) error {

	var wg sync.WaitGroup
	wg.Add(len(w))
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
				Size:       p.metaData.Size,
			}

			p.downloaded = dled
			oldSpeed = speed
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for _, q := range w {
		// This loop keeps trying to download a file if a recoverable error occurs
		go func(v *worker, wgroup *sync.WaitGroup, dl *uint64, cerr, dlerr chan error) {
			begin := v.begin
			end := v.end

			defer func() {

				wgroup.Done()
				cerr <- nil
				dlerr <- nil
			}()

			for {
				downloadPart, err := p.download(begin, end)
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

				d, err := p.copyAt(downloadPart, begin, &downloaded)
				begin += d
				if err != nil {
					if p.verbose {
						log.Printf("error in copying data at offset %d: %v", v.begin, err)
					}
					continue
				}

				if p.verbose {
					fmt.Printf("Copied %d bytes\n", d)
				}

				downloadPart.Close()
				break
			}

		}(q, &wg, &downloaded, errcopy, errdl)
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

// copyAt reads 64 kilobytes from source and copies them to destination at a given offset
func (p *Pluto) copyAt(src io.Reader, offset uint64, dlcounter *uint64) (uint64, error) {
	bufBytes := make([]byte, 256*1024)

	var bytesWritten uint64
	var err error

	for {
		nsr, serr := src.Read(bufBytes)
		if nsr > 0 {
			ndw, derr := p.writer.WriteAt(bufBytes[:nsr], int64(offset))
			if ndw > 0 {
				u64ndw := uint64(ndw)
				offset += u64ndw
				bytesWritten += u64ndw
				atomic.AddUint64(dlcounter, u64ndw)
			}
			if derr != nil {
				err = derr
				break
			}
			if nsr != ndw {
				fmt.Printf("Short write error. Read: %d, Wrote: %d", nsr, ndw)
				err = io.ErrShortWrite
				break
			}
		}

		if serr != nil {
			if serr != io.EOF {
				err = serr
			}
			break
		}
	}

	return bytesWritten, err
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
	p.metaData = fileMetaData{
		Size:               uint64(size),
		Name:               name,
		url:                u,
		MultipartSupported: msupported,
	}
	return nil
}

func (p *Pluto) download(begin, end uint64) (io.ReadCloser, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", p.metaData.url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error in creating GET request: %v", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", begin, end))

	for _, v := range p.headers {
		vsp := strings.Index(v, ":")

		key := v[:vsp]
		value := v[vsp:]

		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {

		if p.verbose {
			fmt.Printf("Requested Bytes %d in range %d-%d. Got %d bytes\n", end-begin, begin, end, resp.ContentLength)
		}

		return nil, fmt.Errorf("error in sending download request: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	if uint64(resp.ContentLength) != (end - begin) {
		return nil, fmt.Errorf(ErrOverflow)
	}
	return resp.Body, nil
}
