FROM google/golang

RUN go get github.com/tools/godep

WORKDIR /gopath/src/gor

ADD . /gopath/src/gor