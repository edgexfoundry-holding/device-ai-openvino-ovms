// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	grpc_client "github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver/grpc-client"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/yiqisoft/mjpeg"
	"gocv.io/x/gocv"
	"google.golang.org/grpc"
)

var once sync.Once
var driver *Driver

// Define Driver struct
type Driver struct {
	lc          logger.LoggingClient
	asyncCh     chan<- *sdkModel.AsyncValues
	grpcConns   map[string]*grpc.ClientConn
	grpcServers map[string]*grpc_client.GRPCInferenceServiceClient
	gocvClients map[string]*gocv.VideoCapture
	mu          sync.Mutex
	writers     map[string]*gocv.VideoWriter
	streams     map[string]*mjpeg.Stream
	imageSizes  map[string]ImageSize
	sdk         interfaces.DeviceServiceSDK
	ovmsCh      map[string]chan OVMSResult
}

// Driver is initialized on service start
func NewProtocolDriver() interfaces.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

// Initialize performs protocol-specific initialization for the device service.
func (d *Driver) Initialize(sdk interfaces.DeviceServiceSDK) error {
	d.lc = sdk.LoggingClient()
	d.asyncCh = sdk.AsyncValuesChannel()

	// init all clients
	d.grpcConns = make(map[string]*grpc.ClientConn)
	d.grpcServers = make(map[string]*grpc_client.GRPCInferenceServiceClient)
	d.gocvClients = make(map[string]*gocv.VideoCapture)
	d.streams = make(map[string]*mjpeg.Stream)
	d.imageSizes = make(map[string]ImageSize)

	d.sdk = sdk
	d.ovmsCh = make(map[string]chan OVMSResult)

	// prepare image matrix

	for _, device := range sdk.Devices() {
		// init stream by device name
		stream := d.NewStreamClient(device.Name)
		http.Handle("/"+device.Name+".mjpeg", stream)
	}

	// initialize HTTP server for streamings
	server := &http.Server{
		Addr: "0.0.0.0:" + LivePort,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			d.lc.Errorf("failed to start HTTP server for live stream: %v", err)
		} else {
			d.lc.Infof("HTTP server started for live streaming on port %s", LivePort)
		}
	}()

	time.Sleep(3 * time.Second)

	// initialize the all devices connection in the service started
	for _, device := range sdk.Devices() {

		var err error
		// init grpc client
		_, err = d.NewGRPCClient(device.Name, device.Protocols)
		if err != nil {
			d.lc.Errorf("failed to initialize GRPC client for '%s' device, skiprotocoling this device: %v", device.Name, err)
			continue
		}
		d.lc.Debugf("GrpcClient connected for device: %s", device.Name)

		// init gocv client
		err = d.NewGocvClient(device.Name, device.Protocols)
		if err != nil {
			d.lc.Errorf("failed to initialize gocv client for '%s' device, skiprotocoling this device: %v", device.Name, err)
			continue
		}
		d.lc.Debugf("GocvClient connected for device: %s", device.Name)
	}

	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) (res []*sdkModel.CommandValue, err error) {
	d.lc.Debugf("Driver.HandleReadCommands: protocols: %v, resource: %v, attributes: %v", protocols, reqs[0].DeviceResourceName, reqs[0].Attributes)

	res = make([]*sdkModel.CommandValue, 0)

	var ovmsResult OVMSResult
	select {
	case ovmsResult = <-d.ovmsCh[deviceName]:
		d.lc.Infof("OVMSResult received, Device: %s, Model: %s, %s fps", deviceName, ovmsResult.ModelName, ovmsResult.InferFPS)
		break

	default:
		// Handle case when channel is send-only
		d.lc.Debugf("Cannot receive from send-only channel d.ovmsCh")
		return nil, nil
	}
	jsonstr, err := json.Marshal(ovmsResult)

	req := reqs[0]
	var cv *sdkModel.CommandValue
	cv, _ = sdkModel.NewCommandValue(req.DeviceResourceName, common.ValueTypeString, string(jsonstr))
	res = append(res, cv)

	return
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource.
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest,
	params []*sdkModel.CommandValue) error {
	d.lc.Debugf("Driver.HandleWriteCommands: protocols: %v, resource: %v, parameters: %v", protocols, reqs[0].DeviceResourceName, params)

	return nil
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if suprotocolorted).
func (d *Driver) Stop(force bool) error {

	d.mu.Lock()
	defer d.mu.Unlock()

	// clear all clients
	for _, client := range d.gocvClients {
		client.Close()
	}
	d.gocvClients = nil

	d.grpcServers = nil

	for _, writer := range d.writers {
		if err := writer.Close(); err != nil {
			d.lc.Errorf("Error closing video writer: %v", err)
		}
	}
	d.writers = nil
	d.streams = nil
	d.imageSizes = nil

	// Then Logging Client might not be initialized
	if d.lc != nil {
		d.lc.Debugf("Driver.Stop called: force=%v", force)
	}
	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (d *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("a new Device is added: %s", deviceName)

	d.mu.Lock()
	defer d.mu.Unlock()

	stream := d.NewStreamClient(deviceName)
	http.Handle("/"+deviceName+".mjpeg", stream)
	// Create new gocv client
	err := d.NewGocvClient(deviceName, protocols)
	if err != nil {
		d.lc.Errorf("failed to initialize gocv client for '%s' device, skiprotocoling this device: %v", deviceName, err)
	}

	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (d *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("Device %s is updated", deviceName)

	// TODO

	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (d *Driver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Debugf("Device %s is removed", deviceName)

	d.mu.Lock()
	defer d.mu.Unlock()
	gocvClient, ok := d.gocvClients[deviceName]
	if ok {
		gocvClient.Close()
	}

	_, ok = d.grpcServers[deviceName]
	if ok {
		delete(d.grpcServers, deviceName)
	}

	// delete http handle
	// http.DefaultServeMux.Handle("/"+deviceName+".mjpeg", http.NotFoundHandler())

	grpc_conn, ok := d.grpcConns[deviceName]
	if ok {
		grpc_conn.Close()
		delete(d.grpcConns, deviceName)
	}

	return nil
}

// Validate device properties
func (d *Driver) ValidateDevice(device models.Device) error {
	d.lc.Debugf("Validating device: %s of profile: %s", device.Name, device.ProfileName)

	var errt error

	// Validate protoco properties
	protocol, ok := device.Protocols[Protocol]
	if !ok {
		errt = fmt.Errorf("'%s' not found in protocols for device '%s'", Protocol, device.Name)
		d.lc.Error(errt.Error())
		return errt
	}

	if err := d.VerifyStringValue(protocol, "Host"); err != nil {
		return err
	}
	if err := d.VerifyNumberValue(protocol, "Port"); err != nil {
		return err
	}
	if err := d.VerifyStringValue(protocol, "Model"); err != nil {
		return err
	}
	if err := d.VerifyNumberValue(protocol, "Version"); err != nil {
		return err
	}
	if err := d.VerifyStringValue(protocol, "Uri"); err != nil {
		return err
	}
	if err := d.VerifyNumberValue(protocol, "Score"); err != nil {
		return err
	}
	if err := d.VerifyBoolValue(protocol, "Record"); err != nil {
		return err
	}
	if err := d.VerifyBoolValue(protocol, "Snapshot"); err != nil {
		return err
	}

	return nil
}

// Driver start
func (d *Driver) Start() error {
	return nil
}

// Discover is called to discover available devices
func (d *Driver) Discover() error {
	return fmt.Errorf("driver's Discover function isn't implemented")
}
