SHELL ?= /bin/bash

export BINARY_NAME ?= opslog
export VERSION ?= $(shell date +%Y%m%d%H%M)
export GOLANG_VERSION ?= 1.12

.DEFAULT_GOAL := package

.PHONY: test
test:
	go test . -v

.PHONY: package
package:
	@docker build -t opslog-builder:${VERSION} \
		--build-arg GOLANG_VERSION=${GOLANG_VERSION} \
		-f Dockerfile .
	docker run -e "GOOS=linux" --rm -v $(shell pwd):/app -w /app opslog-builder:${VERSION} go build -o ${BINARY_NAME} -v
	docker run -e "GOOS=linux" --rm -v $(shell pwd):/app -w /app opslog-builder:${VERSION} go test -v .
	zip -9 -r ../package/${BINARY_NAME}.zip ${BINARY_NAME}
	rm ${BINARY_NAME}