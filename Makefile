
DOCS=./header.apib ./main.apib
DOCFINAL=./build/final.apib
APPNAME=rebblestore-api
LIBNAME:=${APPNAME}.a
TESTNAME=rebblestore-api-tests
SOURCES=$(wildcard *.go) $(wildcard */*.go)
SWAGGER_VERSION=v2.2.8
SWAGGER_UI=https://github.com/swagger-api/swagger-ui/archive/${SWAGGER_VERSION}.tar.gz
SWAGGER_FOLDER=./build/swagger
SWAGGER_TAR=${SWAGGER_FOLDER}/${SWAGGER_VERSION}.tar.gz

.PHONY: all deploy test build doc

all: build doc

${APPNAME}: ${SOURCES}
	go get -v .
	go install -v github.com/mattn/go-sqlite3
	go build -o ${APPNAME} -ldflags "-X main.Buildhost=$(shell hostname -f) -X main.Buildstamp=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p') -X main.Buildgithash=$(shell git rev-parse HEAD)" .

${TESTNAME}: ${SOURCES}
	go get -v github.com/adams-sarah/test2doc/test
	go test -o ${TESTNAME} .

${SWAGGER_TAR}:
	mkdir -p ${SWAGGER_FOLDER}
	wget -r --content-disposition ${SWAGGER_UI} -O ${SWAGGER_TAR}
	tar xvf ${SWAGGER_TAR} -C ${SWAGGER_FOLDER} --strip-components=2 swagger-ui-2.2.8/dist/
	sed 's#http://petstore.swagger.io/v2/swagger.json#http://docs.rebble.io/swagger.json#' -i ${SWAGGER_FOLDER}/index.html

${DOCFINAL}: test ${DOCS} ${APPNAME} ${SWAGGER_TAR}
	mkdir -p ./build/
	cat ${DOCS} > ${DOCFINAL}
	apib2swagger -i ${DOCFINAL} -o ${SWAGGER_FOLDER}/swagger.json
	#docprint -p ${DOCFINAL} -d './build/docs' #-h './build/files/header.html' -c './build/files/custom.css'

deploy:
	sup production deploy

clean:
	rm -v ${LIBNAME} ${APPNAME} ${TESTNAME} || true
	rm -r ${SWAGGER_FOLDER} || true

build: ${APPNAME}
test: ${TESTNAME}
doc: ${DOCFINAL}
