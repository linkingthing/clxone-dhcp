FROM golang:1.14.5-alpine3.12 AS build

ENV GOPROXY=https://goproxy.io

RUN mkdir -p /go/src/github.com/linkingthing/clxone-controller
COPY . /go/src/github.com/linkingthing/clxone-controller

WORKDIR /go/src/github.com/linkingthing/clxone-controller
RUN CGO_ENABLED=0 GOOS=linux go build -o clxone-controller cmd/controller/controller.go

FROM alpine:3.12
COPY --from=build /go/src/github.com/linkingthing/clxone-controller/clxone-controller /

ENTRYPOINT ["/clxone-controller"]
