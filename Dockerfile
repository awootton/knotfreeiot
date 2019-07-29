FROM golang:1.12.1-stretch

ENV PORT 8080
ENV PORT 6161
ENV PORT 1883

RUN go get -u github.com/minio/highwayhash

WORKDIR /go/src/knotfreeiot/

ADD . /go/src/knotfreeiot
RUN go install 

CMD ["/go/bin/knotfree"]