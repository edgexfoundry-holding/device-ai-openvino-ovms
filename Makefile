#
# Copyright (c) 2024 YIQISOFT Ltd
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

.PHONY: build test clean docker unittest lint

ARCH=$(shell uname -m)

MICROSERVICES=cmd/device-ai-openvino-ovms
.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
SDKVERSION=$(shell cat ./go.mod | grep 'github.com/edgexfoundry/device-sdk-go/v3 v' | sed 's/require//g' | awk '{print $$2}')

DOCKER_TAG=$(VERSION)-dev

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-ai-openvino-ovms.Version=$(VERSION) \
                  -X github.com/edgexfoundry/device-sdk-go/v4/internal/common.SDKVersion=$(SDKVERSION)" \
                   -trimpath -mod=readonly
GOTESTFLAGS?=-race

GIT_SHA=$(shell git rev-parse HEAD)

build: $(MICROSERVICES)

build-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging build

tidy:
	go mod tidy

# CGO is enabled by default and cause docker builds to fail due to no gcc,
# but is required for test with -race, so must disable it for the builds only
cmd/device-ai-openvino-ovms:
	CGO_ENABLED=1  go build $(GOFLAGS) -o $@ ./cmd

docker:
	docker build \
		-f Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t hub.yiqisoft.cn/edgexfoundry/device-ai-openvino-ovms:$(GIT_SHA) \
		-t hub.yiqisoft.cn/edgexfoundry/device-ai-openvino-ovms:$(DOCKER_TAG) \
		.

unittest:
	go test $(GOTESTFLAGS) -coverprofile=coverage.out ./...

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run make install-lint"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

install-lint:
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2

test: unittest lint
	go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution-txt.sh

clean:
	rm -f $(MICROSERVICES)

vendor:
	go mod vendor

run: build
	cd cmd && EDGEX_SECURITY_SECRET_STORE=false ./device-ai-openvino-ovms -cp -d -o

push:
	docker buildx build \
	-t hub.yiqisoft.cn/edgexfoundry/device-ai-openvino-ovms:$(DOCKER_TAG) \
	--push \
	--platform linux/amd64,linux/arm64 \
	.
