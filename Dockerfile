FROM google/golang

RUN cd /goroot/src/ && GOOS=linux GOARCH=386 ./make.bash --no-clean

RUN apt-get update && apt-get install ruby vim-common -y

WORKDIR /gopath/src/github.com/buger/gor/

ADD . /gopath/src/github.com/buger/gor/

RUN go get
