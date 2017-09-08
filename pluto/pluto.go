package pluto

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type worker struct {
	begin int64
	end   int64
	url   *url.URL
}

// FileMeta contains information about the file like it's Size, Name and if the server where it is hosted supports multipart downloads
type FileMeta struct {
	Size               int64
	FileName           string
	MultipartSupported bool
}

// Download takes a link and the number of parts that you want to use,
// then downloads the file by dividing it into given number of parts and downloading all parts concurrently.
// If any error occurs in the downloading stage of any part, It'll wait for 2 seconds, Discard the existing part and restart it.
// Discarding whatever bytes were downloaded isn't exactly a smart, So, I'll also be implementing a feature where it can skip over what is already downloaded.
func Download(linkp *url.URL, parts int, verbose bool) (*os.File, error) {

	if linkp == nil {
		return nil, fmt.Errorf("No URL Provided")
	}

	fmeta, err := FetchMeta(linkp)
	if err != nil {
		return nil, fmt.Errorf("error in fetching metadata: %v", err)
	}

	if !fmeta.MultipartSupported {
		parts = 1
	}

	partLimit := fmeta.Size / int64(parts)
	difference := fmeta.Size % int64(parts)

	workers := make([]*worker, parts)

	for i := 0; i < parts; i++ {
		begin := partLimit * int64(i)
		end := partLimit * (int64(i) + 1)

		if i == parts-1 {
			end += difference
		}

		workers[i] = &worker{
			begin: begin,
			end:   end,
			url:   linkp,
		}
	}

	return startDownload(workers, verbose)
}

func startDownload(w []*worker, verbose bool) (*os.File, error) {

	var wg sync.WaitGroup
	wg.Add(len(w))

	completeDownloadFile, err := ioutil.TempFile(os.TempDir(), "pluto_download")
	if err != nil {
		return nil, fmt.Errorf("error in creating temporary file: %v", err)
	}

	errdl := make(chan error, 1)
	errcopy := make(chan error, 1)

	for _, q := range w {
		// This loop keeps trying to download a file if an error occurs
		go func(v *worker, cerr chan error, dlerr chan error) {

			dlPartAbsPath := ""
			for {
				dlPartAbsPath, err = v.download()
				if err != nil {
					if err.Error() == "status code: 400" || err.Error() == "status code: 500" {
						cerr <- err
						return
					}

					if verbose {
						log.Printf("error in downloading a part: %v", err)
					}
					continue
				}

				break
			}

			defer func(p string, w *sync.WaitGroup) {
				errcopy <- nil
				dlerr <- nil
				clean([]string{p})
				w.Done()
			}(dlPartAbsPath, &wg)

			downloadFile, err := ioutil.ReadFile(dlPartAbsPath)
			if err != nil {
				dlerr <- err
				return
			}

			_, err = completeDownloadFile.WriteAt(downloadFile, v.begin)
			if err != nil {
				dlerr <- fmt.Errorf("error in writing a part #%d. File is likely corrupted: %v", v.begin, err)
				return
			}

		}(q, errcopy, errdl)

		err = <-errcopy
		if err != nil {
			return nil, err
		}

		err = <-errdl
		if err != nil {
			return nil, err
		}
	}

	wg.Wait()
	return completeDownloadFile, nil
}

// FetchMeta fetches information about the file like it's Size, Name and if it supports Multipart Download
// If a link does not supports multipart downloads, Then the provided value of part is ignored and set to 1
func FetchMeta(u *url.URL) (*FileMeta, error) {
	resp, err := http.Head(u.String())
	if err != nil {
		return nil, fmt.Errorf("error in sending HEAD request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return nil, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	size := resp.ContentLength
	if size == 0 {
		return nil, fmt.Errorf("Incompatible URL, file size is 0")
	}

	m := false

	if resp.Header.Get("Accept-Range") != "" {
		m = true
	}

	i := strings.LastIndex(u.String(), "/")
	fname := u.String()[i+1:]

	if fname == "" {
		fname = "pluto_download"
	}

	return &FileMeta{Size: size, MultipartSupported: m, FileName: fname}, nil
}

func (w *worker) download() (string, error) {
	downloadFile, err := ioutil.TempFile(os.TempDir(), "pluto_download_part")
	if err != nil {
		return "", err
	}
	defer downloadFile.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", w.url.String(), nil)
	if err != nil {
		return "", fmt.Errorf("error in creating GET request: %v", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", w.begin, w.end))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error in sending download request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	_, err = io.Copy(downloadFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error in copying bytes from http response: %v", err)
	}
	return downloadFile.Name(), nil
}

// Here, p contains the absolute paths to files that needs to be removed
func clean(p []string) {
	// Not handling errors here, Because I used tempfiles everywhere which'll be automatically cleaned anyway
	for _, v := range p {
		os.Remove(v)
	}
}
