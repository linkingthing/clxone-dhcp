GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.6.0
REGISTRY=10.0.0.79:8888

build: clxone_dhcp

clxone_dhcp: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_dhcp cmd/controller/dhcp.go

build-image:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker image prune -f

docker:
	docker build -t linkingthing/clxone-dhcp:${VERSION} .
	docker tag linkingthing/clxone-dhcp:${VERSION} ${REGISTRY}/linkingthing/clxone-dhcp:${VERSION}
	docker push ${REGISTRY}/linkingthing/clxone-dhcp:${VERSION}
	docker image prune -f

clean:
	rm -rf clxone_dhcp

test:
	go test -v -timeout 60s -race ./...

clean-image:
	docker rmi linkingthing/clxone-dhcp:${VERSION}

.PHONY: clean install
