// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	grpc_client "github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver/grpc-client"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/spf13/cast"
	"gocv.io/x/gocv"
)

// imageCapture continuously captures frames from the webcam, performs inference,
// and writes the results to video and stream
func (d *Driver) imageCapture(deviceName string, grpcClient *grpc_client.GRPCInferenceServiceClient, protocols map[string]models.ProtocolProperties) {

	// reconnectInterval is the time to wait before attempting to reconnect to the device,
	// maxReconnectAttempts is the number of times to attempt to reconnect
	reconnectInterval := time.Duration(5) * time.Second
	maxReconnectAttempts := 60

	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		err := d.processMjpegStream(deviceName, grpcClient, protocols)
		if err != nil {
			d.lc.Errorf("Error processing device: %s, error: %v", deviceName, err)
			d.lc.Errorf("Attempting to reconnect in %v seconds (attempt %d/%d)...", reconnectInterval.Seconds(), attempt+1, maxReconnectAttempts)
			time.Sleep(reconnectInterval) // sleep for reconnectInterval seconds
		} else {
			d.lc.Infof("Successfully reconnected to device: %s", deviceName)
			// reset attempt counter
			attempt = 0
			break
		}
	}
	d.lc.Errorf("Maximum number of reconnect attempts reached, exiting...")
}

// processMjpegStream continuously captures frames from the webcam, performs inference, and writes the results to video and stream
func (d *Driver) processMjpegStream(deviceName string, grpcClient *grpc_client.GRPCInferenceServiceClient, protocols map[string]models.ProtocolProperties) error {

	// get parameters from protocols
	d.lc.Debugf("processOutput()")
	protocol := protocols[Protocol]
	score, _ := cast.ToFloat32E(protocol["Score"])
	uri, _ := cast.ToStringE(protocol["Uri"])
	model, _ := cast.ToStringE(protocol["Model"])
	version, _ := cast.ToStringE(protocol["Version"])
	snapshot, _ := cast.ToStringE(protocol["Snapshot"])

	d.ovmsCh[deviceName] = make(chan OVMSResult, 1)

	// define input and output bytes
	var scores []float32
	var tipsColor = color.RGBA{R: 255, G: 0, B: 255, A: 128} // green color
	var fontScale = 1.0
	var fontThinkness = 1
	var fontStyle = gocv.FontHersheyPlain

	// validate score, set default value if invalid
	if score <= 0.0 || score > 1.0 {
		score = 0.6
	}

	cap, err := gocv.VideoCaptureFile(uri)
	if err != nil {
		d.lc.Errorf("Error opening video capture device: %s, error: %v", uri, err)
		return err
	}
	defer cap.Close()

	modelMeata, err := d.ModelMetadataRequest(*grpcClient, model, version)
	if err != nil {
		d.lc.Errorf("Error getting model metadata: %s", err)
		return err
	}
	inputs := modelMeata.GetInputs()
	inputName := inputs[0].Name

	dim := inputs[0].GetShape()
	inputHeight := int(dim[1])
	if inputHeight < 0 {
		inputHeight = 640
	}
	inputWidth := int(dim[2])
	if inputWidth < 0 {
		inputWidth = 640
	}

	for {

		img := gocv.NewMat()
		img_resized := gocv.NewMat()

		scores = scores[:0]

		// invoke inference
		time_start_inference := time.Now()

		if ok := cap.Read(&img); !ok {
			err = fmt.Errorf("failed to read image from MJPEG stream, device: %s", deviceName)
			img.Close()
			img_resized.Close()
			return err
		}

		// skip process if image is empty
		if img.Empty() {
			d.lc.Errorf("Image is empty")
			img.Close()
			img_resized.Close()
			continue
		}
		imgHeight, imgWidth := img.Rows(), img.Cols()
		d.lc.Debugf("Image size: %d x %d", imgWidth, imgHeight)

		// predict image
		inferResponse, err := d.Predict(grpcClient, img, model, version, inputWidth, inputHeight, inputName)
		if err != nil {
			d.lc.Debugf("Error predicting: %s", err)
			img.Close()
			img_resized.Close()
			continue
		}

		time_end_inference := time.Now()
		inferTime := time_end_inference.Sub(time_start_inference)
		d.lc.Debugf("Object Detection Inference time: %s", inferTime)

		modelOutputs := inferResponse.Outputs
		var inferResult = make(map[string][]ObjectDetectionResutl)
		for i, modelOutput := range modelOutputs {
			rawOutputContent := inferResponse.RawOutputContents[i]
			outLen := len(rawOutputContent)
			outShapre := modelOutput.Shape
			outName := modelOutput.Name
			d.lc.Debugf("Inference reuslt, Name: %s, Datatype: %s, Shape: %d, Data len: %d", outName, modelOutput.Datatype, outShapre, outLen)

			inferResult[modelOutput.Name] = d.readData(rawOutputContent, outShapre)
		}

		// write original image to base64 string
		var base64StrOri string = ""
		if snapshot == "true" {
			base64StrOri = "data:image/jpeg;base64,"
			image_bytes, _ := gocv.IMEncode(gocv.JPEGFileExt, img)
			imgBytes := image_bytes.GetBytes()
			base64StrOri += base64.StdEncoding.EncodeToString(imgBytes)
			image_bytes.Close()
		}

		var matched bool = false

		// // draw results on image
		for _, result := range inferResult {

			for _, row := range result {
				inferScore := row.Confidence
				if inferScore > score {
					matched = true
					scores = append(scores, float32(math.Round(float64(inferScore)*100)/100))

					// set face bounding box
					x_min := int(row.X_min * float32(imgWidth))
					if x_min < 0 {
						x_min = 0
					}
					y_min := int(row.Y_min * float32(imgHeight))
					if y_min < 0 {
						y_min = 0
					}
					x_max := int(row.X_max * float32(imgWidth))
					if x_max > imgWidth {
						x_max = imgWidth
					}
					y_max := int(row.Y_max * float32(imgHeight))
					if y_max > imgHeight {
						y_max = imgHeight
					}
					rect := image.Rect(x_min, y_min, x_max, y_max)
					d.lc.Debugf("Cropped face size: %d x %d, rect: %d,%d %d,%d", x_max-x_min, y_max-y_min, x_min, y_min, x_max, y_max)

					// draw rectangle
					gocv.Rectangle(&img, rect, tipsColor, fontThinkness)

					// put label text to image
					score_str := fmt.Sprintf("%.2f", inferScore)
					label_name := coco_classes[int(row.Label)]
					gocv.PutText(
						&img,
						label_name+":"+score_str,
						image.Point{X: x_min, Y: y_min - 5},
						gocv.FontHersheyDuplex,
						fontScale,
						tipsColor,
						fontThinkness)

				}
			}

		}

		if matched {

			// put timestamp text to image
			currentTime := time.Now().Format("2006-01-02 15:04:05")
			gocv.PutText(
				&img,
				currentTime,
				image.Point{X: 10, Y: 30},
				fontStyle,
				fontScale,
				tipsColor,
				fontThinkness)
			infer_second := 1.0 / inferTime.Seconds()
			infer_fps := fmt.Sprintf("%.1f", infer_second)
			d.lc.Debugf("Inference FPS: %s", infer_fps)

			// calculate inference time, put fps text to image
			time_end_process := time.Now()
			inferTime = time_end_process.Sub(time_start_inference)
			infer_second = 1.0 / inferTime.Seconds()
			infer_fps = fmt.Sprintf("%.1f", infer_second)
			gocv.PutText(
				&img, "fps: "+infer_fps,
				image.Point{X: 10, Y: 50},
				fontStyle,
				fontScale,
				tipsColor,
				fontThinkness)
			d.lc.Debugf("Total Inference time: %s, FPS: %s", inferTime, infer_fps)

			// write infer snapshot to base64 string
			base64Str := "data:image/jpeg;base64,"
			image_bytes, err := gocv.IMEncode(gocv.JPEGFileExt, img)
			if err != nil {
				d.lc.Errorf("Error encoding image: %s", err)
				img.Close()
				continue
			}
			imgBytes := image_bytes.GetBytes()
			base64Str += base64.StdEncoding.EncodeToString(imgBytes)

			// write to live stream
			if err := d.WriteStream(deviceName, img); err != nil {
				d.lc.Errorf("Error updating stream: %v", err)
			}

			// write ovms result to channel
			ovmsResult := OVMSResult{
				ModelName: model,
				InferFPS:  infer_fps,
				Snapshot:  base64Str,
				Scores:    scores,
				Original:  base64StrOri,
			}
			ch := d.ovmsCh[deviceName]
			select {
			case ch <- ovmsResult:
			default:
				d.lc.Debugf("OVMS channel is error, drop result.")
			}

			image_bytes.Close()

		}

		// close image Mat
		img.Close()
		img_resized.Close()

	}
}

// read rawOutputContent to ObjectDetectionResutl
func (d *Driver) readData(inputBytes []byte, shape []int64) []ObjectDetectionResutl {
	count := int(shape[2])
	len := int(shape[3])

	var odrs []ObjectDetectionResutl
	f32data := d.readRawDataToFloat32(inputBytes)
	for i := 0; i < count; i++ {
		odr := ObjectDetectionResutl{
			ImageId:    f32data[i*len],
			Label:      f32data[i*len+1],
			Confidence: f32data[i*len+2],
			X_min:      f32data[i*len+3],
			Y_min:      f32data[i*len+4],
			X_max:      f32data[i*len+5],
			Y_max:      f32data[i*len+6],
		}
		odrs = append(odrs, odr)
	}
	return odrs

}
