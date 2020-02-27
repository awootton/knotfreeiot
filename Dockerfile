

# docker build -t gcr.io/fair-theater-238820/knotfreeserver .	
# docker push gcr.io/fair-theater-238820/knotfreeserver


FROM golang:1.14.0-stretch
#FROM golang:1.12-alpine 

# alpine is smaller by 200 MiB but is tragicially git free

ENV PORT 8384
ENV PORT 1883
ENV PORT 7465

WORKDIR /knotfreeiot/

ADD . /knotfreeiot

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

RUN export GO111MODULE=auto; go mod tidy

# RUN ls -lah /go/bin/linux_386
# see knotfreedeploy.yaml # CMD ["/go/bin/linux_386/knotfreeiot"]

RUN export GO111MODULE=auto; go install 
