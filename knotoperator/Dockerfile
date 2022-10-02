# FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

FROM golang:1.14.0-stretch

ENV xxOPERATOR=/usr/local/bin/knotoperator \
    USER_UID=1001 \
    USER_NAME=knotoperator

ENV OPERATOR_NAME=app-operator
ENV WATCH_NAMESPACE=knotspace

ENV HOME=knotoperator

# We can use the 32 bit version to save pointer space?
ENV GOARCH=386

# install operator binary  # from operator build which no longer works
# COPY build/_output/bin/knotoperator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# ENTRYPOINT ["/usr/local/bin/entrypoint"] # from operator build which no longer works

USER ${USER_UID}

WORKDIR /knotdir/

# COPY go.mod .
# COPY go.sum .

# COPY ../iot/go.mod iot/
# COPY ../iot/go.sum iot/

# COPY ../packets/go.mod packets/
# COPY ../packets/go.sum packets/

# COPY ../tokens/go.mod tokens/
# COPY ../tokens/go.sum tokens/

# COPY ../badjson/go.mod badjson/
# COPY ../badjson/go.sum badjson/

# it starts out the same as the knotfree dockerfile

# and then add the code
ADD . /knotdir

# COPY go.mod .
# COPY go.sum .

# COPY iot/go.mod iot/
# COPY iot/go.sum iot/

# COPY packets/go.mod packets/
# COPY packets/go.sum packets/

# COPY tokens/go.mod tokens/
# COPY tokens/go.sum tokens/

# COPY badjson/go.mod badjson/
# COPY badjson/go.sum badjson/

# COPY knotoperator/go.mod knotoperator/
# COPY knotoperator/go.sum knotoperator/

RUN go mod download


# later we: RUN ls -lah /go/bin/linux_386/main.go 
# RUN export GO111MODULE=on; go install knotoperator/main.go 

RUN cd knotoperator;export GO111MODULE=on;go build -o knotoperator

