FROM google/golang

WORKDIR /gopath/src/gor

ADD . /gopath/src/gor

RUN go get