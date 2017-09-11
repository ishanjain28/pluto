PACKAGE=pluto
VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`


LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.Build=${BUILD}"

ARCH_LINUX = $(PACKAGE)-linux-amd64 \
						 $(PACKAGE)-linux-arm64 \
						 $(PACKAGE)-linux-386

ARCH_WIN = $(PACKAGE)-windows-amd64.exe \
					 $(PACKAGE)-windows-386.exe

TARGET = $(PACKAGE)-linux-amd64

# TODO: Must write a configure.ac to find target for user end.
# then we can use:
# all: default
# default: $(TARGET)

all: default
default: $(TARGET)

dist: dist-linux dist-windows

dist-windows: $(ARCH_WIN)
dist-linux: $(ARCH_LINUX)

$(PACKAGE)-linux-arm64:
	GOOS=linux \
	GOARCH=arm64 \
	go build ${LDFLAGS} -o=$@

$(PACKAGE)-linux-amd64:
	GOOS=linux \
	GOARCH=amd64 \
	go build ${LDFLAGS} -o=$@

$(PACKAGE)-linux-386:
	GOOS=linux \
	GOARCH=386 \
	go build ${LDFLAGS} -o=$@

$(PACKAGE)-windows-amd64.exe:
	GOOS=windows \
	GOARCH=amd64 \
	go build ${LDFLAGS} -o=$@

$(PACKAGE)-windows-386.exe:
	GOOS=windows \
	GOARCH=386 \
	go build ${LDFLAGS} -o=$@

dist-clean:
	rm -f $(ARCH_WIN) $(ARCH_LINUX)
