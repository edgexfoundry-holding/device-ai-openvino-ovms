// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2024 YIQISOFT
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// VerifyStringValue validates a string value
func (d *Driver) VerifyStringValue(value models.ProtocolProperties, name string) error {
	var errt error
	strValue, ok := value[name]
	if !ok {
		errt = fmt.Errorf("missing '%s' information", name)
		d.lc.Error(errt.Error())
		return errt
	} else if strValue == "" {
		errt = fmt.Errorf("'%s' is empty, please configure a valid host", name)
		d.lc.Error(errt.Error())
		return errt
	}
	return nil
}

// VerifyNumberValue validates a number value
func (d *Driver) VerifyNumberValue(value models.ProtocolProperties, name string) error {
	var errt error
	numValue, ok := value[name]
	if !ok {
		errt = fmt.Errorf("missing '%s' information", name)
		d.lc.Error(errt.Error())
		return errt
	} else {
		numberString := fmt.Sprintf("%v", numValue)
		_, err := strconv.ParseFloat(numberString, 64)
		if err != nil {
			errt = fmt.Errorf("'%s' must be a number", name)
			d.lc.Error(errt.Error())
			return errt
		}
	}
	return nil
}

// VerifyBoolValue validates a boolean value
func (d *Driver) VerifyBoolValue(value models.ProtocolProperties, name string) error {
	var errt error
	boolValue, ok := value[name]
	if !ok {
		errt = fmt.Errorf("missing '%s' information", name)
		d.lc.Error(errt.Error())
		return errt
	} else if boolValue == "" {
		errt = fmt.Errorf("'%s' is empty, please configure a valid value (true or false)", name)
		d.lc.Error(errt.Error())
		return errt
	}
	boolValueString, ok := boolValue.(string)
	if !ok {
		errt = fmt.Errorf("'%s' must be a string", name)
		d.lc.Error(errt.Error())
		return errt
	}
	_, err := strconv.ParseBool(boolValueString)
	if err != nil {
		errt = fmt.Errorf("invalid value for '%s', please configure a valid boolean value (true or false)", name)
		d.lc.Error(errt.Error())
		return errt
	}
	return nil
}

// Convert slice of 4 bytes to int32 (assumes Little Endian)
func (d *Driver) readFloat32(fourBytes []byte) float32 {
	buf := bytes.NewBuffer(fourBytes)
	var retval float32
	binary.Read(buf, binary.LittleEndian, &retval)
	return retval
}

// read raw bytes data to float32
func (d *Driver) readRawDataToFloat32(inputBytes []byte) []float32 {
	dataLen := len(inputBytes) / 4
	outputData := make([]float32, dataLen)

	for i := 0; i < dataLen; i++ {
		outputData[i] = d.readFloat32(inputBytes[i*4 : i*4+4])
	}
	return outputData
}
