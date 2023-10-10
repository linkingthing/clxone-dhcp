GOSRC = $(shell find . -type f -name '*.go')

VERSION=v2.7.0
go_image=golang:1.18.10-alpine3.16
go_harbor_image=harbor.linkingipam.com/linkingthing/golang-base:1.18.10-alpine3.16

build: clxone_dhcp

clxone_dhcp: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_dhcp cmd/dhcp/dhcp.go

build-cgo: $(GOSRC)
	CGO_ENABLED=1 CGO_CFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_CPPFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_LDFLAGS="-Wl,-z,relro,-z,now" go build -trimpath -buildmode=pie --ldflags '-linkmode=external -extldflags "-Wl,-z,now"' -o clxone_dhcp cmd/dhcp/dhcp.go && strip -s clxone_dhcp 

build-image:
	docker build --build-arg go_image=${go_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f

build-package:
	docker build --build-arg go_image=${go_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker save -o clxone-dhcp-${VERSION}.tar.gz linkingthing/clxone-dhcp:${VERSION}

build-harbor:
	docker build --build-arg go_image=${go_harbor_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker save -o clxone-dhcp-${VERSION}.tar.gz linkingthing/clxone-dhcp:${VERSION}

docker:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker push linkingthing/clxone-dhcp:${VERSION}
	docker image prune -f

clean:
	rm -rf clxone_dhcp

check-static:
	GO111MODULE=on CGO_ENABLED=0 golangci-lint run -v -c .golangci.yml

test:
	go test -v -timeout 60s -race ./...

clean-image:
	docker rmi linkingthing/clxone-dhcp:${VERSION}

.PHONY: clean install
