package pluto_test

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/ishanjain28/pluto/pluto"
)

func TestFetchMeta(t *testing.T) {
	// Incomplete
	u, _ := url.Parse("	")

	resp, err := http.Head(u.String())
	if err != nil {
		log.Printf("error in sending request")
		t.Fail()
		return
	}
	defer resp.Body.Close()

	meta, err := pluto.FetchMeta(u, nil)
	if err != nil {
		log.Println(err.Error())
		t.Fail()
		return
	}
	fmt.Println(meta.Size, resp.ContentLength)

	if meta.Size != uint64(resp.ContentLength) {
		t.Fail()
	}
}

func TestDownload(t *testing.T) {

}
