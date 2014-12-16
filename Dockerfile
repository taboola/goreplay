FROM google/golang

RUN cd /goroot/src/ && GOOS=linux GOARCH=386 ./make.bash --no-clean

RUN apt-get install ruby -y

WORKDIR /gopath/src/gor

ADD . /gopath/src/gor

RUN go get
