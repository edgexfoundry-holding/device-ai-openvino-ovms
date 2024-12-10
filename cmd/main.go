// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides a simple example of an openvino device service.
package main

import (
	device_openvino_face_recog_xdl "github.com/edgexfoundry/device-ai-openvino-ovms"

	"github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/startup"
)

const (
	serviceName string = "device-ai-openvino-ovms"
)

func main() {
	sd := driver.NewProtocolDriver()
	startup.Bootstrap(serviceName, device_openvino_face_recog_xdl.Version, sd)
}
