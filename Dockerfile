# howto:
# docker build -t gcr.  io/fair-theater-238820/knotfreeserver .	
# docker push gcr.   io/fair-theater-238820/knotfreeserver

FROM golang:1.14.0-stretch
#FROM golang:1.12-alpine 
# alpine is smaller by 200 MiB but is tragicially git free

ENV PORT 8384 
ENV PORT 1883
ENV PORT 7465
ENV PORT 8080
ENV PORT 8085
ENV PORT 9090
ENV PORT 3100

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

WORKDIR /knotfreeiot/

##RUN export GO111MODULE=on; go get -u github.com/thei4t/libmqtt@v0.9.9

COPY go.mod .
COPY go.sum .

COPY iot/go.mod iot/
COPY iot/go.sum iot/

COPY packets/go.mod packets/
COPY packets/go.sum packets/

COPY tokens/go.mod tokens/
COPY tokens/go.sum tokens/

COPY badjson/go.mod badjson/
COPY badjson/go.sum badjson/


RUN go mod download

# and then add the code
ADD . /knotfreeiot

# later we: RUN ls -lah /go/bin/linux_386/knotfreeiot # see knotfreedeploy.yaml
RUN export GO111MODULE=on; go install 
