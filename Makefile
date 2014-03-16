all: build-macosx build-x86 build-x64

build-x64:
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
    tar -czf gor_x64.tar.gz gor
    rm gor

build-x86:
    GOOS=linux GOARCH=386 CGO_ENABLED=0 go build
    tar -czf gor_x86.tar.gz gor
    rm gor

build-macosx:
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build
    tar -czf gor_macosx.tar.gz gor
    rm gor