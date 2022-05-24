FROM golang:1.18.1-alpine3.15 AS build

ENV GOPROXY=http://mirrors.aliyun.com/goproxy

RUN mkdir -p /go/src/github.com/linkingthing/clxone-dhcp
COPY . /go/src/github.com/linkingthing/clxone-dhcp

WORKDIR /go/src/github.com/linkingthing/clxone-dhcp
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add build-base
RUN CGO_ENABLED=1 CGO_CFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_CPPFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_LDFLAGS="-Wl,-z,relro,-z,now" go build -trimpath -buildmode=pie --ldflags '-linkmode=external -extldflags "-Wl,-z,now"' -o clxone-dhcp cmd/dhcp/dhcp.go && strip -s clxone-dhcp 

FROM alpine:3.15
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add nmap

COPY --from=build /go/src/github.com/linkingthing/clxone-dhcp/clxone-dhcp /
RUN mkdir -p /opt/files

ENTRYPOINT ["/clxone-dhcp"]
