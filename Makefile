SOURCE = emitter.go gor.go traffic_modifier.go gor_stat.go input_dummy.go input_file.go input_raw.go input_tcp.go limiter.go output_dummy.go output_file.go input_http.go output_http.go output_tcp.go plugins.go settings.go settings_header_filters.go settings_header_hash_filters.go settings_headers.go settings_methods.go settings_option.go settings_url_regexp.go test_input.go elasticsearch.go settings_url_map.go

release: release-x86 release-x64

release-x64:
	docker run -v `pwd`:/gopath/src/gor -t --env GOOS=linux --env GOARCH=amd64 --env CGO_ENABLED=0 -i gor go build && tar -czf gor_x64.tar.gz gor && rm gor

release-x86:
	docker run -v `pwd`:/gopath/src/gor -t --env GOOS=linux --env GOARCH=386 --env CGO_ENABLED=0 -i gor go build && tar -czf gor_x86.tar.gz gor && rm gor

dbuild:
	docker build -t gor .

dtest:
	docker run -v `pwd`:/gopath/src/gor -t -i --env GORACE="halt_on_error=1" gor go test $(ARGS) -race -v --verbose

dfmt:
	docker run -v `pwd`:/gopath/src/gor -t -i gor go fmt

dbench:
	docker run -v `pwd`:/gopath/src/gor -t -i gor go test -v -run NOT_EXISTING -bench HTTP

# Used mainly for debugging, because docker container do not have access to parent machine ports
drun:
	docker run -v `pwd`:/gopath/src/gor -t -i gor go run $(SOURCE) --input-modifier="bash ./examples/echo_modifier.sh" --input-dummy=0 --output-http="http://localhost:9000"  --verbose

dbash:
	docker run -v `pwd`:/gopath/src/gor -t -i gor /bin/bash