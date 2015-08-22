FROM google/golang:1.4

RUN cd /goroot/src/ && GOOS=linux GOARCH=386 ./make.bash --no-clean

RUN apt-get update && apt-get install ruby vim-common -y

# Install Java
# RUN apt-get install -y software-properties-common python-software-properties
# RUN add-apt-repository -y ppa:webupd8team/java
# RUN apt-get update -y
# RUN echo oracle-java8-installer shared/accepted-oracle-license-v1-1 select true | sudo /usr/bin/debconf-set-selections
# RUN apt-get install -y oracle-java8-installer

WORKDIR /gopath/src/github.com/buger/gor/

ADD . /gopath/src/github.com/buger/gor/

RUN go get -u github.com/golang/lint/golint
RUN go get