# howto:
# docker build -t gcr.io/fair-theater-238820/knotfreeserver .	
# docker push gcr.io/fair-theater-238820/knotfreeserver

FROM golang:1.14.0-stretch
#FROM golang:1.12-alpine 
# alpine is smaller by 200 MiB but is tragicially git free

ENV PORT 8384 
ENV PORT 1883
ENV PORT 7465
ENV PORT 8080
ENV PORT 8085
ENV PORT 9090
ENV PORT 3000

WORKDIR /knotfreeiot/

ADD . /knotfreeiot

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

RUN export GO111MODULE=auto; go mod tidy

# RUN ls -lah /go/bin/linux_386/knotfreeiot # see knotfreedeploy.yaml

RUN export GO111MODULE=auto; go install 
