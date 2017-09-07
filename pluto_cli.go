package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ishanjain28/pluto/pluto"
)

func main() {

	u := flag.String("url", "", "Download link of a file")

	parts := flag.Int("part", 16, "Number of Download parts")

	flag.Parse()
	if *u == "" {
		log.Fatalln("no url provided")
	}

	a := time.Now()
	f, err := pluto.Download(*u, *parts)
	if err != nil {
		log.Fatalln(err)
	}

	file, err := os.Create("downloaded_file")
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = io.Copy(file, f)
	if err != nil {
		log.Fatalln(err.Error())
	}
	fmt.Printf("File Downloaded in %s", time.Since(a))
	defer file.Close()
	defer f.Close()

}
