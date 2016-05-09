SOURCE = emitter.go gor.go gor_stat.go input_dummy.go input_file.go input_raw.go input_tcp.go limiter.go output_dummy.go output_file.go input_http.go output_http.go output_tcp.go plugins.go settings.go test_input.go elasticsearch.go http_modifier.go http_modifier_settings.go http_client.go middleware.go protocol.go
SOURCE_PATH = /go/src/github.com/buger/gor/
RUN = docker run -v `pwd`:$(SOURCE_PATH) -p 0.0.0.0:8000:8000 -t -i gor
BENCHMARK = BenchmarkRAWInput
TEST = TestRawListenerBench

release: release-x64

release-x64:
	docker run -v `pwd`:$(SOURCE_PATH) -t --env GOOS=linux --env GOARCH=amd64  -i gor go build -ldflags "-X main.VERSION=$(VERSION) -extldflags \"-static\"" && tar -czf gor_$(VERSION)_x64.tar.gz gor && rm gor

release-x86:
	docker run -v `pwd`:$(SOURCE_PATH) -t --env GOOS=linux --env GOARCH=386 -i gor go build -ldflags "-X main.VERSION=$(VERSION)" && tar -czf gor_$(VERSION)_x86.tar.gz gor && rm gor

build:
	docker build -t gor .


profile:
	go build && ./gor --output-http="http://localhost:9000" --input-dummy 0 --input-raw :9000 --input-http :9000 --memprofile=./mem.out --cpuprofile=./cpu.out --stats --output-http-stats --output-http-timeout 100ms

lint:
	$(RUN) golint $(PKG)

race:
	$(RUN) go test ./... $(ARGS) -v -race -timeout 15s

test:
	$(RUN) go test ./. -timeout 30s $(ARGS) -v

test_all:
	$(RUN) go test ./... -timeout 30s $(ARGS) -v

testone:
	$(RUN) go test ./... -timeout 4s -run $(TEST) $(ARGS) -v

cover:
	$(RUN) go test $(ARGS) -race -v -timeout 15s -coverprofile=coverage.out
	go tool cover -html=coverage.out

fmt:
	$(RUN) gofmt -w -s ./..

vet:
	$(RUN) go vet

bench:
	$(RUN) go test -v -run NOT_EXISTING -bench $(BENCHMARK) -benchtime 5s

profile_test:
	$(RUN) go test $(LDFLAGS) -run $(TEST) ./raw_socket_listener/. $(ARGS) -memprofile mem.mprof -cpuprofile cpu.out
	$(RUN) go test $(LDFLAGS) -run $(TEST) ./raw_socket_listener/. $(ARGS) -c

# Used mainly for debugging, because docker container do not have access to parent machine ports
run:
	$(RUN) go run $(SOURCE) --input-dummy=0 --output-http="http://localhost:9000" --input-raw :9000 --input-http :9000 --verbose --debug --middleware "./examples/middleware/echo.sh"

run-2:
	$(RUN) go run $(SOURCE) --input-file ./fixtures/requests.gor --output-dummy=0

record:
	$(RUN) go run $(SOURCE) --input-dummy=0 --output-file=requests.gor --verbose --debug

replay:
	$(RUN) go run $(SOURCE) --input-file=requests.bin --output-tcp=:9000 --verbose -h

bash:
	$(RUN) /bin/bash
