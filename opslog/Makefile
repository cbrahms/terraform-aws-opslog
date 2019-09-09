SHELL ?= /bin/bash

export BINARY_NAME ?= opslog
export GOLANG_VERSION ?= 1.13

.DEFAULT_GOAL := package

.PHONY: test
test:
	go test . -v

.PHONY: package
package:
	docker run -e "GOOS=linux" --rm -v $(shell pwd):/app -w /app golang:${GOLANG_VERSION} go test -v .
	docker run -e "GOOS=linux" --rm -v $(shell pwd):/app -w /app golang:${GOLANG_VERSION} go build -o ${BINARY_NAME} -v
	zip -9 -r ../package/${BINARY_NAME}.zip ${BINARY_NAME}
	rm ${BINARY_NAME}
