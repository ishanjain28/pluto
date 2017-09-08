build:
	GOOS=linux GOARCH=amd64 go build -o=pluto.linux.amd64
	GOOS=linux GOARCH=386 go build -o=pluto.linux.i386
	GOOS=windows GOARCH=amd64 go build -o=pluto.amd64.exe
	GOOS=windows GOARCH=386 go build -o=pluto.i386.exe

clean:
	rm pluto.linux.i386
	rm pluto.linux.amd64
	rm pluto.i386.exe
	rm pluto.amd64.exe