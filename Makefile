SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY=devcli

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

LDFLAGS=-ldflags "-X github.com/okcredit/devcli.Version=${VERSION} -X github.com/okcredit/devcli.BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build ${LDFLAGS} -o bin/${BINARY} main.go

.PHONY: install
install:
	go install ${LDFLAGS} ./...

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: test
test:
	go test -v ./...

.PHONY: vet
vet:
	go vet ./...
