SOURCE = emitter.go gor.go gor_stat.go input_dummy.go input_file.go input_raw.go input_tcp.go limiter.go output_dummy.go output_file.go input_http.go output_http.go output_tcp.go plugins.go settings.go test_input.go elasticsearch.go http_modifier.go http_modifier_settings.go http_client.go middleware.go protocol.go

SOURCE_PATH = /gopath/src/github.com/buger/gor/

release: release-x86 release-x64

release-x64:
	docker run -v `pwd`:$(SOURCE_PATH) -t --env GOOS=linux --env GOARCH=amd64 --env CGO_ENABLED=0 -i gor go build -ldflags "-X main.VERSION $(VERSION)"&& tar -czf gor_$(VERSION)_x64.tar.gz gor && rm gor

release-x86:
	docker run -v `pwd`:$(SOURCE_PATH) -t --env GOOS=linux --env GOARCH=386 --env CGO_ENABLED=0 -i gor go build -ldflags "-X main.VERSION $(VERSION)" && tar -czf gor_$(VERSION)_x86.tar.gz gor && rm gor

dbuild:
	docker build -t gor .

dlint:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i --env GORACE="halt_on_error=1" gor golint $(PKG)

drace:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i --env GORACE="halt_on_error=1" gor go test ./... $(ARGS) -v -race -timeout 15s

dtest:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go test ./... -timeout 5s $(ARGS) -v

dcover:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i --env GORACE="halt_on_error=1" gor go test $(ARGS) -race -v -timeout 15s -coverprofile=coverage.out
	go tool cover -html=coverage.out

dfmt:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go fmt ./...

dvet:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go vet

dbench:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go test -v -run NOT_EXISTING -bench HTTP

# Used mainly for debugging, because docker container do not have access to parent machine ports
drun:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go run $(SOURCE) --input-dummy=0 --output-http="http://localhost:9000" --input-raw :9000 --input-http :9000 --verbose --debug --middleware "./examples/middleware/echo.sh"

drun-2:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go run $(SOURCE) --input-file="./fixtures/requests.gor" --output-dummy=0 --verbose --debug --middleware "java -cp ./examples/middleware echo"

drecord:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go run $(SOURCE) --input-dummy=0 --output-file=requests.gor --verbose --debug

dreplay:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor go run $(SOURCE) --input-file=requests.bin --output-tcp=:9000 --verbose -h

dbash:
	docker run -v `pwd`:$(SOURCE_PATH) -t -i gor /bin/bash
