package plato

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
	begin          int64
	end            int64
	tmpAbsFilePath string
	url            *url.URL
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

		partAbsFilePath := filepath.Join(saveDir, fmt.Sprintf(".%d.%s", i, fmeta.FileName))

		workers[i] = NewWorker(startFrom, downloadUpto, partAbsFilePath, linkp)
	}

	return Begin(workers)
}

// NewWorker creates a New Worker which is then processed in the Begin Function
func NewWorker(begin, end int64, absFilePath string, u *url.URL) *Worker {
	return &Worker{
		begin:          begin,
		end:            end,
		url:            u,
		tmpAbsFilePath: absFilePath,
	}
}

func Begin(w []*Worker) (io.ReadCloser, error) {

	var wg sync.WaitGroup
	wg.Add(len(w))

	completeDownloadFile, err := ioutil.TempFile(os.TempDir(), "plato_download")
	if err != nil {
		return nil, fmt.Errorf("error in creating temporary file: %v", err)
	}

	for i, worker := range w {
		go func(c int, v *Worker, g *sync.WaitGroup) {
			for {
				downloadFile, err := v.download()
				if err != nil {
					fmt.Printf("an error occurred in downloading part #%d: %v\n", c, err)
					fmt.Println("Retrying...")
					time.Sleep(2 * time.Second)
					continue
				}
				defer downloadFile.Close()
				buf, err := ioutil.ReadAll(downloadFile)
				if err != nil {
					fmt.Printf("failed in reading a part file: %v", err)
					time.Sleep(2 * time.Second)
					continue
				}

				_, err = completeDownloadFile.WriteAt(buf, v.begin)
				if err != nil {
					fmt.Printf("error in writing a part at %d: %v", v.begin, err)
				}

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

func (w *Worker) download() (io.ReadCloser, error) {
	downloadFile, err := ioutil.TempFile(os.TempDir(), "plato_download_part")
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", w.url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", w.begin, w.end))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return nil, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	_, err = io.Copy(downloadFile, resp.Body)
	if err != nil {
		return nil, err
	}
	return downloadFile, nil
}
