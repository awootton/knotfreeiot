FROM golang:1.12.1-stretch
#FROM golang:1.12-alpine # is smaller by 200 MiB

ENV PORT 8080
ENV PORT 6161
ENV PORT 1883

RUN go get -u github.com/minio/highwayhash

WORKDIR /go/src/knotfreeiot/

ADD . /go/src/knotfreeiot

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

RUN go install 

RUN ls -lah /go/bin/linux_386

# see knotfreedeploy.yaml # CMD ["/go/bin/linux_386/knotfreeiot"]