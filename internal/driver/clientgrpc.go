// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

// This package provides an example implementation of
// OpenVINO model server interface.

package driver

import (
	"time"

	grpc_client "github.com/edgexfoundry/device-ai-openvino-ovms/internal/driver/grpc-client"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
)

// Create GRPC client by 'Device' definition
func (d *Driver) NewGRPCClient(deviceName string, protocol map[string]models.ProtocolProperties) (*grpc_client.GRPCInferenceServiceClient, error) {
	d.lc.Debugf("Creating new GRPC client for device %s", deviceName)

	proto := protocol[Protocol]

	host, _ := cast.ToStringE(proto["Host"])
	port, _ := cast.ToStringE(proto["Port"])

	// Connect to gRPC server with retry, default max retry times is 3, retry interval is 3 seconds
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 3; i++ {
		conn, err = grpc.Dial(host+":"+port, grpc.WithInsecure())
		if err == nil {
			break
		}
		d.lc.Errorf("Couldn't connect to endpoint %s: %v", host+":"+port, err)
		time.Sleep(time.Second * 3) // retry interval 3 seconds
	}
	if err != nil {
		d.lc.Errorf("Failed to connect after 3 retries: %v", err)
		return nil, err
	}

	// Create client from gRPC server connection
	client := grpc_client.NewGRPCInferenceServiceClient(conn)
	d.lc.Debugf("Grpc info: %s", client)

	d.grpcConns[deviceName] = conn
	d.grpcServers[deviceName] = &client

	return &client, nil
}

// Get GrpcClient by 'DeviceName'
func (d *Driver) GetGRPCClient(deviceName string, protocols map[string]models.ProtocolProperties) (*grpc_client.GRPCInferenceServiceClient, error) {
	d.lc.Debugf("Getting GRPC client for device: %s", deviceName)

	grpcClient := d.grpcServers[deviceName]

	if grpcClient != nil {
		return grpcClient, nil
	}

	d.lc.Warnf("GRPCClient for device %s not found. Creating it...", deviceName)
	grpcClient, err := d.NewGRPCClient(deviceName, protocols)
	if err != nil {
		return nil, err
	}
	d.grpcServers[deviceName] = grpcClient

	return grpcClient, nil
}
