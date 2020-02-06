VERSION ?= "1.1"
run:
	go run -race -ldflags="-X main.version=${VERSION} -X main.date=$(shell date '+%Y-%m-%dT%H:%M:%S%z')" src/*.go

all: prep binaries docker

prep:
	mkdir -p bin

binaries: linux64 darwin64

build:
	go build main.go

linux64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/ps-opsgenie-grafana64 main.go

darwin64:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/ps-opsgenie-grafanaOSX main.go

pack-linux64: linux64
	upx --brute bin/ps-opsgenie-grafana64

pack-darwin64: darwin64
	upx --brute bin/ps-opsgenie-grafanaOSX

docker: pack-linux64
	docker build --build-arg version="$(VERSION)" -t pasientskyhosting/ps-opsgenie-grafana:latest . && \
	docker build --build-arg version="$(VERSION)" -t pasientskyhosting/ps-opsgenie-grafana:"$(VERSION)" .

docker-run:
	docker run pasientskyhosting/ps-opsgenie-grafana:"$(VERSION)"

docker-push: docker
	docker push pasientskyhosting/ps-opsgenie-grafana:"$(VERSION)"