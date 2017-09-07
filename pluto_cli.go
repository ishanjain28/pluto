package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Printf("Interrupt Detected, Shutting Down.")
		os.Exit(0)
	}()

	a := time.Now()
	f, err := pluto.Download(*u, *parts)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	file, err := os.Create("downloaded_file")
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer file.Close()

	fmt.Printf("File Downloaded in %s", time.Since(a))
	_, err = io.Copy(file, f)
	if err != nil {
		log.Fatalln(err.Error())
	}

}
