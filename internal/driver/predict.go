// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"image"

	grpc_client "github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver/grpc-client"
	"gocv.io/x/gocv"
)

// Predict image using OpenVINO model server
func (d *Driver) Predict(grpcClient *grpc_client.GRPCInferenceServiceClient, img gocv.Mat, model string, version string, width int, height int, inputName string) (*grpc_client.ModelInferResponse, error) {

	// resize image
	img_resized := gocv.NewMat()
	gocv.Resize(img, &img_resized, image.Point{X: width, Y: height}, 0, 0, gocv.InterpolationArea)
	nativeBytes, err := gocv.IMEncode(gocv.JPEGFileExt, img_resized)
	if err != nil {
		d.lc.Errorf("Error encoding image: %s", err)
		nativeBytes.Close()
		img_resized.Close()
		return nil, err
	}
	// get image bytes
	inputBytes := nativeBytes.GetBytes()

	// invoke inference
	inferResponse, err := d.ModelInferRequest(*grpcClient, inputBytes, model, version, inputName)
	if err != nil {
		nativeBytes.Close()
		img_resized.Close()
		return nil, err
	}

	// close resources
	nativeBytes.Close()
	img_resized.Close()

	return inferResponse, nil

}
