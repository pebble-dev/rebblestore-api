
DOCS=./header.apib ./main.apib
DOCFINAL=./build/final.apib
APPNAME=rebblestore-api
LIBNAME:=${APPNAME}.a
TESTNAME=rebblestore-api-tests
SOURCES=$(wildcard *.go) $(wildcard */*.go)

.PHONY: all deploy test build doc

all: build doc

${APPNAME}: ${SOURCES}
	go get -v .
	go install -v github.com/mattn/go-sqlite3
	go build -o ${APPNAME} -ldflags "-X main.Buildhost=$(shell hostname -f) -X main.Buildstamp=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p') -X main.Buildgithash=$(shell git rev-parse HEAD)" .

${TESTNAME}: ${SOURCES}
	go get -v github.com/adams-sarah/test2doc/test
	go test -o ${TESTNAME} .

${DOCFINAL}: test ${DOCS} ${APPNAME}
	mkdir -p ./build/
	cat ${DOCS} > ${DOCFINAL}
	#docprint -p ${DOCFINAL} -d './build/docs' #-h './build/files/header.html' -c './build/files/custom.css'

deploy:
	sup production deploy

clean:
	rm -v ${LIBNAME} ${APPNAME} ${TESTNAME} || true

build: ${APPNAME}
test: ${TESTNAME}
doc: ${DOCFINAL}
