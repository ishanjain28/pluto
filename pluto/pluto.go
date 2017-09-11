package pluto

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

type worker struct {
	begin uint64
	end   uint64
	url   *url.URL
}

// FileMeta contains information about the file like it's Size and if the server where it is hosted supports multipart downloads
type FileMeta struct {
	u                  *url.URL
	Size               uint64
	Name               string
	MultipartSupported bool
}

// Config contains all the details that Download needs.
// RetryCount is not used at this point.
// Parts is the number of connections to use to download a file
// Verbose is to enable verbose mode.
// Writer is the place where downloaded data is written.
type Config struct {
	Parts      uint
	Verbose    bool
	Writer     io.WriterAt
	Meta       *FileMeta
	RetryCount int
}

// Download takes Config struct
// then downloads the file by dividing it into given number of parts and downloading all parts concurrently.
// If any error occurs in the downloading stage of any part, It'll check if the the program can recover from error by retrying download
// And if an error occurs which the program can not recover from, it'll return that error
func Download(c *Config) error {

	// Limit number of CPUs it can use
	runtime.GOMAXPROCS(runtime.NumCPU() / 2)
	// If server does not supports, Set parts to 1
	if !c.Meta.MultipartSupported {
		c.Parts = 1
	}

	perPartLimit := c.Meta.Size / uint64(c.Parts)
	difference := c.Meta.Size % uint64(c.Parts)

	workers := make([]*worker, c.Parts)

	var i uint
	for i = 0; i < c.Parts; i++ {
		begin := perPartLimit * uint64(i)
		end := perPartLimit * (uint64(i) + 1)

		if i == c.Parts-1 {
			end += difference
		}

		workers[i] = &worker{
			begin: begin,
			end:   end,
			url:   c.Meta.u,
		}
	}

	return startDownload(workers, c.Verbose, c.Writer)
}

func startDownload(w []*worker, verbose bool, writer io.WriterAt) error {

	var wg sync.WaitGroup
	wg.Add(len(w))
	var err error

	errdl := make(chan error, 1)
	errcopy := make(chan error, 1)

	count := len(w)

	for _, q := range w {
		// This loop keeps trying to download a file if a recoverable error occurs
		go func(v *worker, wgroup *sync.WaitGroup, cerr, dlerr chan error) {
			begin := v.begin
			end := v.end

			defer func() {
				count--

				wgroup.Done()
				cerr <- nil
				dlerr <- nil
			}()

			for {
				downloadPart, err := download(begin, end, v.url)
				if err != nil {
					if err.Error() == "status code: 400" || err.Error() == "status code: 500" {
						cerr <- err
						return
					}

					if verbose {
						log.Println(err)
					}
					continue
				}

				d, err := copyAt(writer, downloadPart, begin)
				if err != nil {
					if verbose {
						log.Printf("error in copyAt at offset %d: %v", v.begin, err)
					}
					begin += d
					continue
				}

				if verbose {
					fmt.Printf("Copied %d bytes\n", d)
				}

				downloadPart.Close()
				begin += d
				break
			}

		}(q, &wg, errcopy, errdl)
	}

	if verbose {
		go func(w []*worker) {
			for {
				fmt.Println("Connections Active", count)
				time.Sleep(3 * time.Second)
			}
		}(w)
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
func copyAt(dst io.WriterAt, src io.Reader, offset uint64) (uint64, error) {
	bufBytes := make([]byte, 64*1024)

	var bytesWritten int64
	var err error

	for {
		nsr, serr := src.Read(bufBytes)
		if nsr > 0 {
			ndw, derr := dst.WriteAt(bufBytes[:nsr], int64(offset))
			if ndw > 0 {
				offset += uint64(ndw)
				bytesWritten += int64(ndw)
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

	return uint64(bytesWritten), err
}

// FetchMeta fetches information about the file like it's Size, Name and if it supports Multipart Download
// If a link does not supports multipart downloads, Then the provided value of part is ignored and set to 1
func FetchMeta(u *url.URL) (*FileMeta, error) {
	resp, err := http.Head(u.String())
	if err != nil {
		return nil, fmt.Errorf("error in sending HEAD request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	size := resp.ContentLength
	if size == 0 {
		return nil, fmt.Errorf("Incompatible URL, file size is 0")
	}

	m := true

	if resp.Header.Get("Accept-Range") != "" && resp.Header.Get("Accept-Ranges") != "" {
		m = false
	}

	resp, err = http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error in sending GET request: %v", err)
	}

	cDispose := strings.Split(resp.Header.Get("Content-Disposition"), "filename=")

	name := ""

	if len(cDispose) > 0 {
		cdfilename := cDispose[1]
		cdfilename = cdfilename[1:]
		cdfilename = cdfilename[:len(cdfilename)-1]
		name = cdfilename
	}

	resp.Body.Close()

	return &FileMeta{Size: uint64(size), Name: name, u: u, MultipartSupported: m}, nil
}

func download(begin, end uint64, u *url.URL) (io.ReadCloser, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error in creating GET request: %v", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", begin, end))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error in sending download request: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
