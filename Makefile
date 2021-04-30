GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.4.1

build: clxone_dhcp

clxone_dhcp: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_dhcp cmd/controller/controller.go

build-image:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f

docker:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f
	docker push linkingthing/clxone-dhcp:${VERSION}

clean:
	rm -rf clxone_dhcp

test:
	go test -v -timeout 60s -race ./...

clean-image:
	docker rmi linkingthing/clxone-dhcp:${VERSION}

.PHONY: clean install
