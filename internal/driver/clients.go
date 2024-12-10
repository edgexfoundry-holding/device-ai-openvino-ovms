// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"fmt"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/yiqisoft/mjpeg"
	"gocv.io/x/gocv"
)

// Create Stream Writer by device name
func (d *Driver) NewStreamClient(deviceName string) *mjpeg.Stream {
	d.lc.Debugf("Creating new Stream client for device %s", deviceName)

	stream := mjpeg.NewStream()
	d.streams[deviceName] = stream
	return stream
}

// Write live stream to client by device name
func (d *Driver) WriteStream(deviceName string, img gocv.Mat) error {
	stream, ok := d.streams[deviceName]
	if !ok {
		return fmt.Errorf("stream for device %s not found", deviceName)
	}

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
	if err != nil {
		return err
	}
	stream.UpdateJPEG(buf.GetBytes())
	defer buf.Close()

	return nil
}

// Get GocvClient by 'DeviceName'
func (d *Driver) GetGocvClient(deviceName string, protocols map[string]models.ProtocolProperties) (*gocv.VideoCapture, error) {
	d.lc.Debugf("Getting Gocv client for device: %s", deviceName)

	gocvClient := d.gocvClients[deviceName]

	// var webcam *gocv.VideoCapture
	if gocvClient == nil {
		d.lc.Warnf("GocvClient for device %s not found. Creating it...", deviceName)
		err := d.NewGocvClient(deviceName, protocols)
		if err != nil {
			delete(d.gocvClients, deviceName)
			return nil, err
		}
		d.gocvClients[deviceName] = gocvClient
	} else {
		d.lc.Debugf("GocvClient exist.")
	}

	return gocvClient, nil

}

// Create RTSP client by 'Device' definition
func (d *Driver) NewGocvClient(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Debugf("Creating new Gocv client for device %s", deviceName)

	// get gRPC client
	grpcClient, err := d.GetGRPCClient(deviceName, protocols)
	if err != nil {
		err = fmt.Errorf("failed to get grpc client for device: %s", deviceName)
		return err
	}

	go func() {
		d.imageCapture(deviceName, grpcClient, protocols)
	}()

	return err
}

// 91 [Common Objects in Context (COCO)](https://cocodataset.org/#home) dataset
var coco_classes = []string{
	"__background__", "person", "bicycle", "car", "motorcycle", "airplan", "bus", "train", "car", "boat",
	"traffic light", "fire hydrant", "street sign", "stop sign", "parking meter", "bench", "bird", "cat",
	"dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "hat", "backpack", "umbrella",
	"shoe", "eye glasses", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball",
	"kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle", "plate",
	"wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange", "broccoli",
	"carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch", "potted plant", "bed", "mirror",
	"dining table", "window", "desk", "toilet", "door", "tv", "laptop", "mouse", "remote", "keyboard",
	"cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "blender", "book", "clock",
	"vase", "scissors", "teddy bear", "hair drier", "toothbrush", "hair brush",
}
