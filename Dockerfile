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


ARG BASE=golang:1.23-alpine3.20
FROM ${BASE} AS builder

ARG MAKE=make build

WORKDIR /device-openvino

LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2024: YIQISOFT'

RUN apk add --update --no-cache make git build-base musl-dev alpine-sdk cmake clang clang-dev make gcc g++ libc-dev linux-headers unzip autoconf libtool protobuf libprotobuf opencv-dev

WORKDIR /device-openvino
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Compile API
# RUN wget https://raw.githubusercontent.com/openvinotoolkit/model_server/releases/2023/3/src/kfserving_api/grpc_predict_v2.proto
RUN echo 'option go_package = "./grpc-client";' >> grpc_predict_v2.proto
RUN protoc --go_out="./" --go-grpc_out="./" ./grpc_predict_v2.proto

# Compile go app
COPY go.mod vendor* ./
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .
RUN ${MAKE}

# Next image - Copy built Go binary into new workspace
FROM alpine:3.20
LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2024: YIQISOFT'

RUN apk add --update --no-cache dumb-init libc-dev opencv
# Ensure using latest versions of all installed packages to avoid any recent CVEs
RUN apk --no-cache upgrade

WORKDIR /
COPY --from=builder /device-openvino/cmd/Attribution.txt /Attribution.txt
COPY --from=builder /device-openvino/cmd/device-ai-openvino-ovms /device-ai-openvino-ovms
COPY --from=builder /device-openvino/cmd/res /res

EXPOSE 61805

ENTRYPOINT ["/device-ai-openvino-ovms"]
CMD ["-cp=keeper.http://edgex-core-keeper:59890", "--registry"]