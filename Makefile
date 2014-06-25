#Copyright (c) 2014 Ashley Jeffs
#
#Permission is hereby granted, free of charge, to any person obtaining a copy
#of this software and associated documentation files (the "Software"), to deal
#in the Software without restriction, including without limitation the rights
#to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
#copies of the Software, and to permit persons to whom the Software is
#furnished to do so, subject to the following conditions:
#
#The above copyright notice and this permission notice shall be included in
#all copies or substantial portions of the Software.
#
#THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
#OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
#THE SOFTWARE.

.PHONY: all fmt build vet lint check clean install

#build: export GOOS=linux
#build: export GOARCH=amd64

all: clean check install

build: clean
	@go build

fmt:
	@gofmt -w ./$*

vet:
	@VET=`go tool vet ./**/*.go`; if [ ! -z "$$VET" ]; then echo "$$VET"; fi; test -z "$$VET";

lint:
	@LINT=`golint ./**/*.go`; if [ ! -z "$$LINT" ]; then echo "$$LINT"; fi; test -z "$$LINT";

check: fmt vet lint
	@go test -v ./...
	@cd leapclient; \
		find . -maxdepth 1 -name "test_*" -exec nodeunit {} \;

clean:
	@find $(GOPATH)/pkg/*/github.com/jeffail -name leaps.a -delete

install: clean
	@go install
