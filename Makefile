SOURCE = emitter.go gor.go gor_stat.go input_dummy.go input_file.go input_raw.go input_tcp.go limiter.go output_dummy.go output_file.go output_http.go output_tcp.go plugins.go settings.go settings_header_filters.go settings_header_hash_filters.go settings_headers.go settings_methods.go settings_option.go settings_url_regexp.go test_input.go elasticsearch.go settings_url_map.go

all: build-x86 build-x64

build-x64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
	tar -czf gor_x64.tar.gz gor
	rm gor

build-x86:
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build
	tar -czf gor_x86.tar.gz gor
	rm gor

dbuild:
	docker build -t gor .

dtest:
	docker run -v `pwd`:/gopath/src/gor -t -i gor go test -v

drun:
	docker run -v `pwd`:/gopath/src/gor -t -i gor go run $(SOURCE) --input-dummy=0 --output-dummy=0 --verbose

dbash: 
	docker run -v `pwd`:/gopath/src/gor -t -i gor /bin/bash