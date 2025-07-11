ARG go_image
ARG base_image

FROM $go_image AS build
ARG git_commit
ARG build_time
ARG build_version

ENV GOPROXY=https://goproxy.cn,direct

RUN mkdir -p /go/src/github.com/linkingthing/clxone-dhcp
COPY . /go/src/github.com/linkingthing/clxone-dhcp

WORKDIR /go/src/github.com/linkingthing/clxone-dhcp
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add build-base
ENV GOSUMDB=off
RUN rm -f go.sum
RUN go mod tidy
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" \
    CGO_CPPFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" \
    CGO_LDFLAGS="-Wl,-z,relro,-z,now" \
    go build -trimpath -buildmode=pie \
    --ldflags "-linkmode=external -extldflags \"-Wl,-z,now\" -X main.version=${build_version} -X main.commit=${git_commit} -X main.buildTime=${build_time}" \
    -o clxone-dhcp cmd/dhcp/dhcp.go && strip -s clxone-dhcp

FROM $base_image
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add nmap
RUN apk add libcap

RUN sed -i "/sync/d" /etc/passwd
RUN sed -i "/sync/d" /etc/shadow
RUN sed -i "/shutdown/d" /etc/passwd
RUN sed -i "/shutdown/d" /etc/shadow
RUN sed -i "/halt/d" /etc/passwd
RUN sed -i "/halt/d" /etc/shadow
RUN sed -i "/operator/d" /etc/passwd
RUN sed -i "/operator/d" /etc/shadow
RUN rm -rf etc/ssl/certs/ca-certificates.crt

COPY --from=build /go/src/github.com/linkingthing/clxone-dhcp/clxone-dhcp /
RUN mkdir -p /opt/files
RUN setcap CAP_NET_BIND_SERVICE=+eip /clxone-dhcp

ENTRYPOINT ["/clxone-dhcp"]
