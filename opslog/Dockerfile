ARG GOLANG_VERSION
ARG VERSION
ARG IMAGE
FROM golang:${GOLANG_VERSION} AS dep_downloader

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM golang:${GOLANG_VERSION}

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOARCH=amd64

VOLUME /app 
WORKDIR /app

COPY --from=dep_downloader /go/pkg /go/pkg