// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"context"
	"time"

	grpc_client "github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver/grpc-client"
)

// Make inference request
func (d *Driver) ModelInferRequest(client grpc_client.GRPCInferenceServiceClient, img []byte, modelName string, modelVersion string, inputName string) (*grpc_client.ModelInferResponse, error) {

	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contents := grpc_client.InferTensorContents{}
	contents.BytesContents = append(contents.BytesContents, img)

	// prep inference request
	inferInput := grpc_client.ModelInferRequest_InferInputTensor{
		Name:     inputName,
		Datatype: "BYTES",
		Shape:    []int64{1},
		Contents: &contents,
	}

	// Create request input tensors
	inferInputs := []*grpc_client.ModelInferRequest_InferInputTensor{
		&inferInput,
	}

	// Create inference request for specific model/version
	modelInferRequest := grpc_client.ModelInferRequest{
		ModelName:    modelName,
		ModelVersion: modelVersion,
		Inputs:       inferInputs,
	}

	// Submit inference request to server
	modelInferResponse, err := client.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		d.lc.Debugf("Error processing InferRequest: %v", err)
		return nil, err
	}
	return modelInferResponse, nil
}

func (d *Driver) ModelMetadataRequest(client grpc_client.GRPCInferenceServiceClient, modelName string, modelVersion string) (*grpc_client.ModelMetadataResponse, error) {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	modelMetadataRequest := grpc_client.ModelMetadataRequest{
		Name:    modelName,
		Version: modelVersion,
	}
	// Submit modelMetadata request to server
	modelMetadataResponse, err := client.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		d.lc.Errorf("Error processing Metadata: %v", err)
		return nil, err
	}
	return modelMetadataResponse, nil
}

// Make inference request
func (d *Driver) ModelInferRequestFP32(client grpc_client.GRPCInferenceServiceClient, img []float32, modelName string, modelVersion string, inputName string) (*grpc_client.ModelInferResponse, error) {

	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// contents := grpc_client.InferTensorContents{}
	// contents.BytesContents = append(contents.BytesContents, img)
	// contents.Fp32Contents = append(contents.Fp32Contents, img)
	// inputData := make([]float32, len(img)/4)
	// for i := 0; i < len(img)/4; i++ {
	// 	inputData[i] = d.readFloat32(img[i*4 : i*4+4])
	// }

	// prep inference request
	inferInput := grpc_client.ModelInferRequest_InferInputTensor{
		Name:     inputName,
		Datatype: "FP32",
		Shape:    []int64{1, 112, 112, 3},
		Contents: &grpc_client.InferTensorContents{
			Fp32Contents: img,
		},
	}

	// Create request input tensors
	inferInputs := []*grpc_client.ModelInferRequest_InferInputTensor{
		&inferInput,
	}

	// Create inference request for specific model/version
	modelInferRequest := grpc_client.ModelInferRequest{
		ModelName:    modelName,
		ModelVersion: modelVersion,
		Inputs:       inferInputs,
	}

	// Submit inference request to server
	modelInferResponse, err := client.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		d.lc.Debugf("Error processing InferRequest: %v", err)
		return nil, err
	}
	return modelInferResponse, nil
}
