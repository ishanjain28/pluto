package pluto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
)

type worker struct {
	begin   uint64
	end     uint64
	url     *url.URL
	writer  io.WriterAt
	headers []string
	verbose bool
	ctx     context.Context
}

// copyAt reads 64 kilobytes from source and copies them to destination at a given offset
func (w *worker) copyAt(src io.Reader, dlcounter *uint64) (uint64, error) {
	bufBytes := make([]byte, 256*1024)

	var bytesWritten uint64
	var err error

	for {
		nsr, serr := src.Read(bufBytes)
		if nsr > 0 {
			ndw, derr := w.writer.WriteAt(bufBytes[:nsr], int64(w.begin))
			if ndw > 0 {
				u64ndw := uint64(ndw)
				w.begin += u64ndw
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

func (w *worker) download() (io.ReadCloser, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", w.url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error in creating GET request: %v", err)
	}
	req = req.WithContext(w.ctx)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", w.begin, w.end))

	for _, v := range w.headers {
		vsp := strings.Index(v, ":")

		key := v[:vsp]
		value := v[vsp:]

		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {

		if w.verbose {
			fmt.Printf("Requested Bytes %d in range %d-%d. Got %d bytes\n", w.end-w.begin, w.begin, w.end, resp.ContentLength)
		}

		return nil, fmt.Errorf("error in sending download request: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	if uint64(resp.ContentLength) != (w.end - w.begin) {
		return nil, fmt.Errorf(ErrOverflow)
	}
	return resp.Body, nil
}
