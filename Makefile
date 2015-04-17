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

.PHONY: all build lint check clean install example multiplat package

PROJECT := leaps
JS_PATH := ./client
JS_CLIENT := $(JS_PATH)/leapclient.js

BIN := ./bin
JS_BIN := $(BIN)/js

VERSION := $(shell git describe --tags || echo "v0.0.0")
DATE := $(shell date +"%c" | tr ' :' '__')

GOFLAGS := -ldflags "-X github.com/jeffail/util.version $(VERSION) \
	-X github.com/jeffail/util.dateBuilt $(DATE)"

help:
	@echo "Leaps build system, run one of the following commands:"
	@echo ""
	@echo "    make build    : Build the service and generate client libraries"
	@echo ""
	@echo "    make lint     : Run linting on both .go and .js files"
	@echo "    make check    : Run unit tests on both Golang and JavaScript code"
	@echo ""
	@echo "    make package  : Package the service, scripts and client libraries"
	@echo "                    into a .tar.gz archive for all supported operating"
	@echo "                    systems"
	@echo ""
	@echo "    make clean    : Clean the repository of any built/generated files"


build: check
	@mkdir -p $(JS_BIN)
	@echo ""; echo " -- Building $(BIN)/$(PROJECT) -- ";
	@go build -o $(BIN)/$(PROJECT) $(GOFLAGS)
	@cp $(BIN)/$(PROJECT) $$GOPATH/bin
	@echo "copying/compressing js libraries into $(JS_BIN)"
	@cat $(JS_CLIENT) $(JS_PATH)/leap-bind-*.js > $(JS_BIN)/leaps.js; \
		cat $(JS_PATH)/LICENSE > "$(JS_BIN)/leaps-min.js"; \
		uglifyjs "$(JS_BIN)/leaps.js" >> "$(JS_BIN)/leaps-min.js";

GOLINT=$(shell golint .)
lint:
	@echo ""; echo " -- Linting Golang and JavaScript files -- ";
	@gofmt -w . && go tool vet ./**/*.go && echo "$(GOLINT)" && test -z "$(GOLINT)" && jshint $(JS_PATH)/*.js

check: lint
	@echo ""; echo " -- Unit testing Golang and JavaScript files -- ";
	@go test ./...
	@cd $(JS_PATH); find . -maxdepth 1 -name "test_*" -exec nodeunit {} \;
	@echo ""; echo " -- Testing complete -- ";

clean:
	@find $(GOPATH)/pkg/*/github.com/jeffail -name $(PROJECT).a -delete
	@rm -rf $(BIN)

PLATFORMS = "darwin/amd64/" "freebsd/amd64/" "freebsd/arm/7" "freebsd/arm/5" "linux/amd64/" "linux/arm/7" "linux/arm/5" "windows/amd64/"

multiplatform_builds = $(foreach platform, $(PLATFORMS), \
		plat="$(platform)" armspec="$${plat\#*/}" \
		GOOS="$${plat%/*/*}" GOARCH="$${armspec%/*}" GOARM="$${armspec\#*/}"; \
		bindir="$(BIN)/$${GOOS}_$${GOARCH}$${GOARM}" exepath="$${bindir}/bin/$(PROJECT)"; \
		echo "building $${exepath} with GOOS=$${GOOS}, GOARCH=$${GOARCH}, GOARM=$${GOARM}"; \
		mkdir -p "$${bindir}/bin"; \
		GOOS=$$GOOS GOARCH=$$GOARCH GOARM=$$GOARM go build -o "$$exepath" $(GOFLAGS); \
	)

multiplat: build
	@echo ""; echo " -- Building multiplatform binaries -- ";
	@$(multiplatform_builds)

package_builds = $(foreach platform, $(PLATFORMS), \
		plat="$(platform)" armspec="$${plat\#*/}" \
		GOOS="$${plat%/*/*}" GOARCH="$${armspec%/*}" GOARM="$${armspec\#*/}"; \
		p_stamp="$${GOOS}_$${GOARCH}$${GOARM}" a_name="$(PROJECT)-$${p_stamp}"; \
		echo "archiving $${a_name} version $(VERSION)"; \
		mkdir -p "./releases/$(VERSION)"; \
		cp -LR "$(BIN)/$${p_stamp}" "./releases/$(VERSION)/$(PROJECT)"; \
		cp -LR "$(BIN)/js" "./releases/$(VERSION)/$(PROJECT)"; \
		cp -LR "./config" "./releases/$(VERSION)/$(PROJECT)"; \
		cp -LR "./static" "./releases/$(VERSION)/$(PROJECT)"; \
		cp -LR "./scripts" "./releases/$(VERSION)/$(PROJECT)"; \
		cp -LR "./docs" "./releases/$(VERSION)/$(PROJECT)"; \
		cd "./releases/$(VERSION)"; \
		tar -czf "$${a_name}.tar.gz" "./$(PROJECT)"; \
		rm -r "./$(PROJECT)"; \
		cd ../..; \
	)

package: multiplat
	@echo ""; echo " -- Packaging multiplatform archives -- ";
	@$(package_builds)
