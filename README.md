# pluto
A CLI and Library for lightning fast, aggressive and reliable downloads. 

Pluto is a Multipart File Downloader. It comes in form of a package and a CLI. It works by dividing the file into a given number of parts, Each part is given a range of bytes to download, Once all the downloads are complete, it stiches all of them together in correct order to generate the file.

There are a lot of tool similar and better than Pluto but most of them have an upper limit of 16 or 32 parts whereas Pluto has no upper limit.

[![GoDoc](https://godoc.org/github.com/ishanjain28/pluto/pluto?status.svg)](https://godoc.org/github.com/ishanjain28/pluto/pluto) 
[![Go Report Card](https://goreportcard.com/badge/github.com/ishanjain28/pluto)](https://goreportcard.com/report/github.com/ishanjain28/pluto)

## Installation

1. You have a working Go Environment

    go get github.com/ishanjain28/pluto

2. See the [Releases](https://github.com/ishanjain28/pluto/releases) section for Precompiled Binaries

### CLI Example

	pluto --help 
	Usage of pluto:
		-part int
        		Number of Download parts (default 16)



	pluto --part=10 [urls ...]


### Package Example:

    package main

    import (
	    "flag"
	    "io"
	    "log"
	    "os"

        "github.com/ishanjain28/pluto/pluto"
    )

    func main() {

	    u := flag.String("url", "", "Download link of a file")

	    parts := flag.Int("part", 16, "Number of Download parts")

	    flag.Parse()
	    if *u == "" {
    		log.Fatalln("no url provided")
	    }

	    f, err := pluto.Download(*u, *parts)
	    if err != nil {
		    log.Fatalln(err)
    	}
    	defer f.Close()
	    // A copy of completed file is saved in Temp Directory, It is usually deleted automatically
	    // But you can do so manually if you want
	    defer os.Remove(f.Name())

	    file, err := os.Create("downloaded_file")
	    if err != nil {
    		log.Fatalln(err.Error())
	    }

	    defer file.Close()

	    _, err = io.Copy(file, f)
	    if err != nil {
            log.Fatalln(err.Error())
	    }
    }




## Default Behaviours

1. When an error occurs in downloading stage of a part, It is automatically retried, unless there is an error that retrying won't fix. For example, If the server sends a 404, 400 or 500 HTTP response, It stop and return an error.

2. To keep RAM usage to a minimum, One file is created for each part in temporary directory. All the data downloaded is then copied to these files. When all the parts finish downloading, A new temporary file is created and data from all different parts is written to this file and a pointer to it is returned.

3. When a part download fails for reason that is recoverable(see 1) reason, All the data downloaded until the point of error is discarded and then that part is redownloaded.


## Motivation

Almost all Download Managers have an upper limit on number of parts. This is usually done to for the following reasons:

1. Prevent DDoS detection systems on servers from falsely marking the client's IP address as a hostile machine.
2. Prevent internet experience degradtion on other Machines on the same local network and in other applications on the same PC.
3. This is just a guess, But maybe People saw that after a certain limit increasing number of parts doesn't really increase anymore speed which is true but the 16/32 part limit is very low and much better speed can be achieved by increasing part limit upto 100 on a 50Mbit Connection.

But when I am downloading a file from my private servers I need the absolute maximum speed and I could not find a good tool for it. So, I built one myself. A benchmark b/w Pluto, axel and aria2c will be added shortly. 


##### Please use this package responsibly because it can cause all the bad things mentioned above

# License 

MIT