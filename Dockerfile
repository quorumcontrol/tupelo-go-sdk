FROM golang:1.11.5

WORKDIR /go/src/github.com/quorumcontrol/tupelo-go-client

COPY . .

RUN go build ./...
