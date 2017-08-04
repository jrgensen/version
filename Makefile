NAME=version
REPO=registry.blackwoodseven.com/businesslogic/$(NAME)
WORKDIR=/go/src/$(NAME)
DOCKER=docker run --rm -ti -v `pwd`:/go -w $(WORKDIR) --env CGO_ENABLED=0 golang:1.9

compile: dependencies
	$(DOCKER) go build -a -installsuffix cgo .

build: compile
	docker build -t $(REPO) .

push:
	docker push $(REPO)

watch: dependencies
	$(DOCKER) ginkgo watch

dependencies:
	test -s bin/ginkgo || ( $(DOCKER) go get github.com/onsi/ginkgo/ginkgo; )
	$(DOCKER) ginkgo bootstrap || true;
	$(DOCKER) go get -t ./...
	
test: dependencies
	$(DOCKER) go test ./...

fmt:
	$(DOCKER) go fmt ./...


run:
	docker run -d -p 8080:80 -v /var/run/docker.sock:/var/run/docker.sock --name $(NAME) -t $(REPO)

stop:
	docker rm -f $(NAME)

rerun: stop run

.PHONY: compile build watch dependencies test init
