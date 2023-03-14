# howto:
# docker build -t gcr.  io/fair-theater-238820/knotfreeserver .	
# docker push gcr.   io/fair-theater-238820/knotfreeserver

# bitnami/minideb:bullseye is a debian 11 image 78.5MB

FROM golang:1.19.0-alpine

RUN apk add lsof

ENV PORT 8384 
ENV PORT 1883
ENV PORT 7465
ENV PORT 8080
ENV PORT 8085
ENV PORT 9090
ENV PORT 3100

# We can use the 32 bit version to save pointer space
ENV GOARCH=386

WORKDIR /knotfreeiot/

COPY go.mod .
COPY go.sum .

RUN go mod download  && go mod verify

# and then add the code
ADD . /knotfreeiot

# later we: RUN /go/bin/linux_386/knotfreeiot # see knotfreedeploy.yaml
#  RUN export GO111MODULE=on; go install 

RUN CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -o manager main.go
