/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dbManager

import (
	"errors"
	"os/exec"
	"strings"

	common "IEdgeInsights/InfluxDBConnector/common"
	util "IEdgeInsights/libs/common/go"
	inflxUtil "IEdgeInsights/libs/common/influxdb"

	"github.com/golang/glog"
)

// InfluxDBManager structure
type InfluxDBManager struct {
	CnInfo common.AppConfig
	DbInfo common.DbCredential
	//subInfoList []common.SubScriptionInfo
}

// Init will start the InfluxDb server and create a user
func (idbMgr *InfluxDBManager) Init() error {

	var cmd *exec.Cmd

	if idbMgr.CnInfo.DevMode {
		cmd = exec.Command("./InfluxDBConnector/influx_start.sh", "dev_mode")
	} else {
		cmd = exec.Command("./InfluxDBConnector/influx_start.sh")
	}

	err := cmd.Run()
	if err != nil {
		glog.Errorf("Failed to start influxdb Server, Error: %s", err)
		return err
	}

	portUp := util.CheckPortAvailability(idbMgr.DbInfo.Host, idbMgr.DbInfo.Port)
	if !portUp {
		glog.Error("Influx DB port not up")
		return errors.New("Influx DB port not up")
	}
	clientAdmin, err := inflxUtil.CreateHTTPClient(idbMgr.DbInfo.Host, idbMgr.DbInfo.Port, "", "", idbMgr.CnInfo.DevMode)
	if err != nil {
		glog.Errorf("Error creating InfluxDB client: %v", err)
		return err
	}
	resp, err := inflxUtil.CreateAdminUser(clientAdmin, idbMgr.DbInfo.Username, idbMgr.DbInfo.Password, idbMgr.DbInfo.Database)

	if err == nil && resp.Error() == nil {
		glog.Infof("Successfully created admin user: %s", idbMgr.DbInfo.Username)
	} else {
		if resp != nil && resp.Error() != nil {
			glog.Infof("admin user already exists")
		} else {
			glog.Errorf("Error code: %v while creating "+"admin user: %s", err, idbMgr.DbInfo.Username)
		}
	}
	clientAdmin.Close()

	return nil
}

// CreateDataBase will create a database in InfluxDb
func (idbMgr *InfluxDBManager) CreateDataBase(dbName string, retention string) error {
	// Create InfluxDB database
	glog.Infof("Creating InfluxDB database: %s", idbMgr.DbInfo.Database)
	client, err := inflxUtil.CreateHTTPClient(idbMgr.DbInfo.Host,
		idbMgr.DbInfo.Port, idbMgr.DbInfo.Username, idbMgr.DbInfo.Password, idbMgr.CnInfo.DevMode)

	if err != nil {
		glog.Errorf("Error creating InfluxDB client: %v", err)
		return err
	}
	defer client.Close()
	response, err := inflxUtil.CreateDatabase(client, dbName, retention)
	if err != nil {
		glog.Errorf("Cannot create database: %s", response.Error())
		return err
	}

	if err == nil && response.Error() == nil {
		glog.Infof("Successfully created database: %s", dbName)
	} else {
		if response.Error() != nil {
			glog.Errorf("Error code: %v, Error Response: %s while creating "+
				"database: %s", err, response.Error(), dbName)
		} else {
			glog.Errorf("Error code: %v while creating "+"database: %s", err, dbName)
		}
		return err
	}

	return nil
}

// Subscribe func subscribes to InfluxDB and starts up the udp server
func (idbMgr *InfluxDBManager) Subscribe(subInfo common.SubScriptionInfo, out common.OutPutInterface) error {

	//Setup the subscription for the DB
	// We have one DB only to be used by DA. Hence adding subscription
	// only during inititialization.

	client, err := inflxUtil.CreateHTTPClient(idbMgr.DbInfo.Host,
		idbMgr.DbInfo.Port, idbMgr.DbInfo.Username, idbMgr.DbInfo.Password, idbMgr.CnInfo.DevMode)

	if err != nil {
		glog.Errorf("Error creating InfluxDB client: %v", err)
		return err
	}

	defer client.Close()

	response, err := inflxUtil.DropAllSubscriptions(client, idbMgr.DbInfo.Database)
	if err != nil {
		glog.Errorln("Error in dropping subscriptions")
		return err
	}

	subscriptionName := subInfo.DbName + "Subscription"
	response, err = inflxUtil.CreateSubscription(client, subscriptionName,
		subInfo.DbName, subInfo.Host, subInfo.Port, idbMgr.CnInfo.DevMode)

	var InfluxSC InfluxSubCtx
	InfluxSC.SbInfo = subInfo
	InfluxSC.OutInterface = out

	if err == nil && response.Error() == nil {
		glog.Infoln("Successfully created subscription")
		go InfluxSC.startServer(idbMgr.CnInfo.DevMode)
	} else if response.Error() != nil {
		glog.Errorf("Response error: %v while creating subscription", response.Error())
		const str = "already exists"

		// TODO: we need to handle this situation in a more better way in
		// future in cases when DataAgent dies abruptly, system reboots etc.,
		if strings.Contains(response.Error().Error(), str) {
			glog.Infoln("subscription already exists, let's start the UDP" +
				" server anyways..")
			go InfluxSC.startServer(idbMgr.CnInfo.DevMode)
		}
	}

	return nil
}

// GetAttribute func will return the measurement name from the data
func (idbMgr *InfluxDBManager) GetAttribute(data []byte) (string, error) {
	point := strings.Split(string(data), ",")
	if len(point) > 1 {
		return point[0], nil
	}

	err := errors.New("Empty String")
	return point[0], err
}
