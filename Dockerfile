FROM golang:1.14.5-alpine3.12 AS build

ENV GOPROXY=https://goproxy.io

RUN mkdir -p /go/src/github.com/linkingthing/clxone-dhcp
COPY . /go/src/github.com/linkingthing/clxone-dhcp

WORKDIR /go/src/github.com/linkingthing/clxone-dhcp
RUN CGO_ENABLED=0 GOOS=linux go build -o clxone-dhcp cmd/dhcp/dhcp.go

FROM alpine:3.12
COPY --from=build /go/src/github.com/linkingthing/clxone-dhcp/clxone-dhcp /
RUN mkdir -p /opt/files

ENTRYPOINT ["/clxone-dhcp"]
