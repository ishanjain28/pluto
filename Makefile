PACKAGE=pluto
VERSION=1.4

ARCH_LINUX = $(PACKAGE)-$(VERSION)-linux-amd64 \
						 $(PACKAGE)-$(VERSION)-linux-arm64 \
						 $(PACKAGE)-$(VERSION)-linux-386

ARCH_WIN = $(PACKAGE)-$(VERSION)-windows-amd64.exe \
					 $(PACKAGE)-$(VERSION)-windows-386.exe

TARGET = $(PACKAGE)-$(VERSION)-linux-amd64

# TODO: Must write a configure.ac to find target for user end.
# then we can use:
# all: default
# default: $(TARGET)

all: default
default: $(TARGET)

dist: dist-linux dist-windows

dist-windows: $(ARCH_WIN)
dist-linux: $(ARCH_LINUX)

$(PACKAGE)-$(VERSION)-linux-arm64:
	GOOS=linux \
	GOARCH=arm64 \
	go build -o=$@

$(PACKAGE)-$(VERSION)-linux-amd64:
	GOOS=linux \
	GOARCH=amd64 \
	go build -o=$@

$(PACKAGE)-$(VERSION)-linux-386:
	GOOS=linux \
	GOARCH=386 \
	go build -o=$@

$(PACKAGE)-$(VERSION)-windows-amd64.exe:
	GOOS=windows \
	GOARCH=amd64 \
	go build -o=$@

$(PACKAGE)-$(VERSION)-windows-386.exe:
	GOOS=windows \
	GOARCH=386 \
	go build -o=$@

dist-clean:
	rm -f $(ARCH_WIN) $(ARCH_LINUX)
