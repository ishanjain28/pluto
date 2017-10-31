package pluto_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/ishanjain28/pluto/pluto"
)

func setupHTTPServer() *http.Server {
	srv := &http.Server{Addr: "127.0.0.1:5050"}

	http.HandleFunc("/testfile", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "fixtures/testfile")
	})
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Printf("error setingup the httpServer: %v", err)
		}
	}()
	return srv
}

func TestMain(m *testing.M) {
	srv := setupHTTPServer()

	retCode := m.Run()
	srv.Shutdown(nil)

	os.Exit(retCode)
}

func TestFetchMeta(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:5050/testfile")

	resp, err := http.Head(u.String())
	if err != nil {
		t.Fatalf("error sending request: %v", err)

	}
	defer resp.Body.Close()
	p, err := pluto.New(u, []string{}, 1, false)
	if err != nil {
		t.Fatalf("unable to create pluto instance: %v", err)
	}

	if p.MetaData.Size != uint64(resp.ContentLength) {
		t.Fatalf("fetched metadata size does not match with the response.ContentLength")
	}
}

func TestDownload(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:5050/testfile")
	p, err := pluto.New(u, []string{}, 1, false)
	if err != nil {
		t.Fatalf("unable to create pluto instance: %v", err)
	}
	f, err := ioutil.TempFile("/tmp", "pluto")
	if err != nil {
		t.Fatalf("unable to create temp file: %v", err)
	}
	defer f.Close()
	r, err := p.Download(context.Background(), f)
	if err != nil {
		t.Fatalf("unable to download file: %v", err)
	}
	t.Logf("Result: %v\n", r)
	download, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("unable to read downloaded file: %v", err)
	}
	original, err := ioutil.ReadFile("fixtures/testfile")
	if err != nil {
		t.Fatalf("unable to read original file")
	}
	if !reflect.DeepEqual(download, original) {
		t.Errorf("downloaded file and original file are not equal!")
	}
}
