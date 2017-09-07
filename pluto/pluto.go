package pluto

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Worker struct {
	begin int64
	end   int64
	url   *url.URL
}

type FileMeta struct {
	Size               int64
	FileName           string
	MultipartSupported bool
}

func Download(link string, parts int) (io.ReadCloser, error) {

	if link == "" {
		return nil, fmt.Errorf("No URL Provided")
	}

	linkp, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("error in parsing url: %v", err)
	}

	fmeta, err := FetchMeta(linkp)
	if err != nil {
		return nil, err
	}

	partLimit := fmeta.Size / int64(parts)
	difference := fmeta.Size % int64(parts)

	h := md5.New()
	h.Write([]byte(linkp.String()))
	saveDir := filepath.Join(os.TempDir(), fmt.Sprintf("%x", string(h.Sum(nil))))
	os.MkdirAll(saveDir, 0x777)

	workers := make([]*Worker, parts)

	for i := 0; i < parts; i++ {
		startFrom := partLimit * int64(i)
		downloadUpto := partLimit * (int64(i) + 1)

		if i == parts-1 {
			downloadUpto += difference
		}

		workers[i] = NewWorker(startFrom, downloadUpto, linkp)
	}

	return Begin(workers)
}

// NewWorker creates a New Worker which is then processed in the Begin Function
func NewWorker(begin, end int64, u *url.URL) *Worker {
	return &Worker{
		begin: begin,
		end:   end,
		url:   u,
	}
}

func Begin(w []*Worker) (io.ReadCloser, error) {

	var wg sync.WaitGroup
	wg.Add(len(w))

	completeDownloadFile, err := ioutil.TempFile(os.TempDir(), "plato_download")
	if err != nil {
		return nil, fmt.Errorf("error in creating temporary file: %v", err)
	}

	// TODO: Write a cleanup function that erases all remaining parts

	for i, worker := range w {
		go func(c int, v *Worker, g *sync.WaitGroup) {
			for {
				n, err := v.download()
				if err != nil {
					fmt.Printf("an error occurred in downloading part #%d: %v\n", c, err)
					fmt.Println("Retrying...")
					time.Sleep(2 * time.Second)
					continue
				}

				downloadFile, err := ioutil.ReadFile(n)
				if err != nil {
					fmt.Printf("failed in reading a part file: %v\n", err)
					time.Sleep(2 * time.Second)
					continue
				}

				_, err = completeDownloadFile.WriteAt(downloadFile, v.begin)
				if err != nil {
					fmt.Printf("error in writing a part at %d: %v\n", v.begin, err)
				}
				defer clean([]string{n})
				wg.Done()
				break
			}
		}(i, worker, &wg)
	}
	wg.Wait()

	return completeDownloadFile, nil
}

func FetchMeta(u *url.URL) (*FileMeta, error) {
	resp, err := http.Head(u.String())
	if err != nil {
		return nil, err
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
		fname = "plato_download"
	}

	return &FileMeta{Size: size, MultipartSupported: m, FileName: fname}, nil
}

func (w *Worker) download() (string, error) {
	downloadFile, err := ioutil.TempFile(os.TempDir(), "plato_download_part")
	if err != nil {
		return "", err
	}
	defer downloadFile.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", w.url.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", w.begin, w.end))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return "", fmt.Errorf("status code is %d", resp.StatusCode)
	}

	_, err = io.Copy(downloadFile, resp.Body)
	if err != nil {
		return "", err
	}
	return downloadFile.Name(), nil
}

func clean(a []string) {
	// Not handling errors here, Because I used tempfiles everywhere which'll be automatically cleaned anyway
	for _, v := range a {
		os.Remove(v)
	}
}
