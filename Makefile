GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.4.1

build: clxone_controller

clxone_controller: $(GOSRC)
	CGO_ENABLED=0 GOOS=linux go build -o clxone_controller cmd/controller/controller.go

build-image:
	docker build -t linkingthing/clxone-controller:${VERSION} .
	docker image prune -f

docker:
	docker build -t linkingthing/clxone-controller:${VERSION} .
	docker image prune -f
	docker push linkingthing/clxone-controller:${VERSION}

clean:
	rm -rf clxone_controller

test:
	go test -v -timeout 60s -race ./...

clean-image:
	docker rmi linkingthing/clxone-controller:${VERSION}

.PHONY: clean install
