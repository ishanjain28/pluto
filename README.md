# pluto
A CLI and Library for lightning fast, aggressive and reliable downloads. 

Pluto is a Multipart File Downloader. It comes in form of a package and a CLI. It works by dividing the file into a given number of parts, Each part is given a range of bytes to download, As all the parts are downloading they are also written to the saveFile in correct order.

There are a lot of tool similar and better than Pluto but most of them have an upper limit of 16 or 32 parts whereas Pluto has no upper limit.

[![GoDoc](https://godoc.org/github.com/ishanjain28/pluto/pluto?status.svg)](https://godoc.org/github.com/ishanjain28/pluto/pluto) 
[![Go Report Card](https://goreportcard.com/badge/github.com/ishanjain28/pluto)](https://goreportcard.com/report/github.com/ishanjain28/pluto)
[![Build Status](https://travis-ci.org/ishanjain28/pluto.svg?branch=master)](https://travis-ci.org/ishanjain28/pluto)

## Installation

1. You have a working Go Environment

    go get github.com/ishanjain28/pluto

2. See the [Releases](https://github.com/ishanjain28/pluto/releases) section for Precompiled Binaries

### CLI Example

	Usage of pluto:
  		-name string
    		Path or Name of save File
  		-part uint
    	  	Number of Download parts (default 32)
  		-verbose
    		Enable Verbose Mode


	$ pluto [OPTIONS] [URLs...]


### Package Example:

	See pluto_cli.go for an example of this package



## Default Behaviours

1. When an error occurs in downloading stage of a part, It is automatically retried, unless there is an error that retrying won't fix. For example, If the server sends a 404, 400 or 500 HTTP response, It stop and return an error.

2. To keep RAM usage to a minimum, Only 64kilobytes of data is read at a time from HTTP connection and written to file.

3. When a part download fails for reason that is recoverable(see 1) reason, Only the bytes that have not been downloaded yet are requested from server.


## Motivation

Almost all Download Managers have an upper limit on number of parts. This is usually done to for the following reasons:

1. Prevent DDoS detection systems on servers from falsely marking the client's IP address as a hostile machine.
2. Prevent internet experience degradtion on other Machines on the same local network and in other applications on the same PC.
3. This is just a guess, But maybe People saw that after a certain limit increasing number of parts doesn't really increase anymore speed which is true but the 16/32 part limit is very low and much better speed can be achieved by increasing part limit upto 100 on a 50Mbit Connection.

But when I am downloading a file from my private servers I need the absolute maximum speed and I could not find a good tool for it. So, I built one myself. A benchmark b/w Pluto, axel and aria2c will be added shortly. 


##### Please use this package responsibly because it can cause all the bad things mentioned above and if you encounter any problems, Feel free to create an issue.

# License 

GPLv2