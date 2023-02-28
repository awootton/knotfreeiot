
# howto:
# docker build -t gcr.io/fair-theater-238820/monitor_pod .	
# docker push gcr.io/fair-theater-238820/monitor_pod

# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

RUN go mod download

# Copy the go source
COPY main/main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# later we: RUN /workspace/manager # see deploy.yaml
RUN export GO111MODULE=on; go install 
