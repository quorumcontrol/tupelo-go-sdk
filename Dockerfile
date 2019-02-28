FROM golang:1.12.0

WORKDIR /go/src/github.com/quorumcontrol/tupelo-go-client


COPY . .

RUN go build ./...
