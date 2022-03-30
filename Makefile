GOSRC = $(shell find . -type f -name '*.go')

VERSION=v2.0.3

build: clxone_dhcp

clxone_dhcp: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_dhcp cmd/dhcp/dhcp.go

build-image:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f

build-package:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker save -o clxone-dhcp-${VERSION}.tar.gz linkingthing/clxone-dhcp:${VERSION}

docker:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker push linkingthing/clxone-dhcp:${VERSION}
	docker image prune -f

clean:
	rm -rf clxone_dhcp

test:
	go test -v -timeout 60s -race ./...

clean-image:
	docker rmi linkingthing/clxone-dhcp:${VERSION}

.PHONY: clean install
