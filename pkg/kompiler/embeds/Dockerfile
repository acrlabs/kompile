FROM golang:1.23-bookworm

RUN go install github.com/go-delve/delve/cmd/dlv@latest

COPY main /main

CMD ["/main"]
