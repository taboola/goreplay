FROM google/golang

RUN cd /goroot/src/ && GOOS=linux GOARCH=386 ./make.bash --no-clean

WORKDIR /gopath/src/github.com/buger/gor/

ADD . /gopath/src/github.com/buger/gor/

RUN go get