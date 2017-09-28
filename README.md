# pluto
A CLI and Library for lightning fast, aggressive and reliable downloads. 

Pluto is a Multipart File Downloader. It comes in form of a package and a CLI. It works by dividing the file into a given number of parts, Each part is given a range of bytes to download, As all the parts are downloading they are also written to the file in correct order.

There are a lot of tool similar and better than Pluto but most of them have an upper limit of 16 or 32 parts whereas Pluto has no upper limit.

[![GoDoc](https://godoc.org/github.com/ishanjain28/pluto/pluto?status.svg)](https://godoc.org/github.com/ishanjain28/pluto/pluto) 
[![Go Report Card](https://goreportcard.com/badge/github.com/ishanjain28/pluto)](https://goreportcard.com/report/github.com/ishanjain28/pluto)
[![Build Status](https://travis-ci.org/ishanjain28/pluto.svg?branch=master)](https://travis-ci.org/ishanjain28/pluto)

## Features

1. Fast download speed.
2. Multi Part Downloading
3. High tolerance for low quality internet connection. 
4. A Stats API that makes getting parameters like current download speed and number of bytes downloaded easier
5. Guarantees reliable file downloads
6. It can be used to download files from servers which require authorization in form of a value in Request Header.
7. It can load URLs from a file.


## Installation

1. You have a working Go Environment

	go get github.com/ishanjain28/pluto

2. You don't have a working Go Environment 
   1. See the [Releases](https://github.com/ishanjain28/pluto/releases) section for Precompiled Binaries
   2. Download a binary for your platform
   3. Put the binary in `/usr/bin` or `/usr/local/bin` on Unix like systems and add the path to binary to `PATH` variable on Windows.
   4. Done. Now type `pluto -v` in terminal to see if it is installed correctly. :)


### CLI Example
	Usage:
		pluto [OPTIONS] [urls...]

	Application Options:
	      --verbose         Enable Verbose Mode
	  -n, --connections=    Number of concurrent connections
	      --name=           Path or Name of save file
	  -f, --load-from-file= Load URLs from a file
	  -H, --Headers=        Headers to send with each request. Useful if a server requires some information in headers
	  -v, --version         Print Pluto Version and exit

	Help Options:
	  -h, --help            Show this help message

### Package Example:

	See cli.go for an example of this package


## Default Behaviours

1. When an error occurs in downloading stage of a part, It is automatically retried, unless there is an error that retrying won't fix. For example, If the server sends a 404, 400 or 500 HTTP response, It stop and return an error.

2. It now uses 256kb buffers instead of a 64kb buffer to reduce CPU Usage. 

3. When a part download fails for reason that is recoverable(see 1) reason, Only the bytes that have not been downloaded yet are requested from server.


## Motivation

Almost all Download Managers have an upper limit on number of parts. This is usually done to for the following reasons:

1. Prevent DDoS detection systems on servers from falsely marking the client's IP address as a hostile machine.
2. Prevent internet experience degradtion on other Machines on the same local network and in other applications on the same PC.
3. This is just a guess, But maybe People saw that after a certain limit increasing number of parts doesn't really increase anymore speed which is true but the 16/32 part limit is very low and much better speed can be achieved by increasing part limit upto 100 on a 50Mbit Connection.

But when I am downloading a file from my private servers I need the absolute maximum speed and I could not find a good tool for it. So, I built one myself. A benchmark b/w Pluto, axel and aria2c will be added shortly. 

## Future Plans

1. Pause and resume support.
2. Intelligent redistribution of remaining bytes when one of the connections finishes downloading data assigned to it. This would result in much better speed utilisation as it approaches the end of download.


##### Please use this package responsibly because it can cause all the bad things mentioned above and if you encounter any problems, Feel free to create an issue.

# License 

GPLv2