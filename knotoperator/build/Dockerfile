# FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

FROM golang:1.14.0-stretch

ENV OPERATOR=/usr/local/bin/knotoperator \
    USER_UID=1001 \
    USER_NAME=knotoperator

ENV OPERATOR_NAME=app-operator
ENV WATCH_NAMESPACE=knotspace

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

# install operator binary
# COPY build/_output/bin/knotoperator ${OPERATOR}

# COPY build/bin /usr/local/bin
# RUN  /usr/local/bin/user_setup

# ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

WORKDIR /knotoperator/

RUN ls

COPY go.mod .
COPY go.sum .

RUN go mod download

# and then add the code
ADD . /knotoperator

# later we: RUN ls -lah /go/bin/linux_386/main.go 
RUN export GO111MODULE=on; go install cmd/manager/main.go

