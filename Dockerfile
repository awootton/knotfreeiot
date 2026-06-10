# howto:
# docker build -t gcr.  io/fair-theater-238820/knotfreeserver .	
# docker push gcr.   io/fair-theater-238820/knotfreeserver

# bitnami/minideb:bullseye is a debian 11 image 78.5MB

#FROM golang:1.19.0-alpine
FROM golang:1.21-alpine

RUN apk add lsof

ENV PORT 8384 
ENV PORT 1883
ENV PORT 7465
ENV PORT 8080
ENV PORT 8085
ENV PORT 9090
ENV PORT 3100

# We can use the 32 bit version to save pointer space
# ENV GOARCH=386

WORKDIR /knotfreeiot/

COPY go.mod .
COPY go.sum .

RUN go mod download  && go mod verify

# and then add the code
ADD . /knotfreeiot

# should we use the smaller memory model when not CGO and race?
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -race -o  manager main.go

# no race detector: 
RUN CGO_ENABLED=0 GOOS=linux go build -a -o manager main.go

# with race detector:
# add this to enable cgo for race detector, otherwise it will fail
# Install GCC and required dependencies
# RUN apk add --no-cache build-base
# RUN CGO_ENABLED=1 GOOS=linux go build -a -race -o manager main.go
