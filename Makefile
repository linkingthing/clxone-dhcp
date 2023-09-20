GOSRC = $(shell find . -type f -name '*.go')

VERSION=v2.0.10
# build amd64
go_image=golang:1.18.10-alpine3.16
base_image=alpine:3.16
go_harbor_image=harbor.linkingipam.com/linkingthing/golang-base:1.18.10-alpine3.16
# build arm64
go_arm_image=golang:1.18.10-alpine3.17
base_arm_image=alpine:3.17

build: clxone_dhcp

clxone_dhcp: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_dhcp cmd/dhcp/dhcp.go

build-cgo: $(GOSRC)
	CGO_ENABLED=1 CGO_CFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_CPPFLAGS="-fstack-protector-all -ftrapv -D_FORTIFY_SOURCE=2 -O2" CGO_LDFLAGS="-Wl,-z,relro,-z,now" go build -trimpath -buildmode=pie --ldflags '-linkmode=external -extldflags "-Wl,-z,now"' -o clxone_dhcp cmd/dhcp/dhcp.go && strip -s clxone_dhcp 

build-image:
	docker build --build-arg go_image=${go_image} --build-arg base_image=${base_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f

build-package:
	docker build --platform linux/amd64 --build-arg go_image=${go_image} --build-arg base_image=${base_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker save -o clxone-dhcp-${VERSION}.tar.gz linkingthing/clxone-dhcp:${VERSION}

build-package-arm:
	docker build --platform linux/arm64 --build-arg go_image=${go_arm_image} --build-arg base_image=${base_arm_image} -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker save -o clxone-dhcp-${VERSION}.tar.gz linkingthing/clxone-dhcp:${VERSION}

build-harbor:
	docker build --build-arg go_image=${go_harbor_image} --build-arg base_image=${base_image} -t linkingthing/clxone-dhcp:${VERSION} .
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
