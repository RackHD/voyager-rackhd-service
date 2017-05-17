ORGANIZATION = RackHD
PROJECT = voyager-rackhd-service
BINARYNAME = voyager-rackhd-service
GOOUT = ./bin
export RABBITMQ_URL = amqp://guest:guest@localhost:5672

default: deps build test

deps:
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega
	go get github.com/gorilla/mux
	go get github.com/satori/go.uuid
	go get ./...

integration-test:
	ginkgo -r -race -trace -cover -randomizeAllSpecs --slowSpecThreshold=30 --focus="\bINTEGRATION\b" -v

unit-test:
	ginkgo -r -race -trace -cover -randomizeAllSpecs --slowSpecThreshold=30 --focus="\bUNIT\b" -v

test:
	ginkgo -r -race -trace -cover

cover-cmd: test
	go tool cover -html=cmd/cmd.coverprofile

build:
	go build -o $(GOOUT)/$(BINARYNAME)
