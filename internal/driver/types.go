// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

type OVMSInfo struct {
	Host     string
	Port     int
	Model    string
	Version  int
	Uri      string
	Snapshot bool
	Record   bool
	Score    string
}

type ImageSize struct {
	Width  int
	Height int
}

type Coordinate struct {
	X_min int `json:"x_min"`
	Y_min int `json:"y_min"`
	X_max int `json:"x_max"`
	Y_max int `json:"y_max"`
}

// OVMSResult struct for inference result
type OVMSResult struct {
	ModelName string    `json:"model_name"`
	InferFPS  string    `json:"infer_fps"`
	Snapshot  string    `json:"snapshot"`
	Scores    []float32 `json:"scores"`
	Original  string    `json:"original"`
}

type ObjectDetectionResutl struct {
	ImageId    float32 `json:"image_id"`
	Label      float32 `json:"label"`
	Confidence float32 `json:"confidence"`
	X_min      float32 `json:"x_min"`
	Y_min      float32 `json:"y_min"`
	X_max      float32 `json:"x_max"`
	Y_max      float32 `json:"y_max"`
}

type ObjectDetectionResutls struct {
	Results []ObjectDetectionResutl `json:"results"`
}
