/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	"flag"
	"os"
	
	eiscfgmgr "ConfigMgr/eisconfigmgr"
	eismsgbus "EISMessageBus/eismsgbus"
	common "IEdgeInsights/InfluxDBConnector/common"
	configManager "IEdgeInsights/InfluxDBConnector/configmanager"
	dbManager "IEdgeInsights/InfluxDBConnector/dbmanager"
	pubManager "IEdgeInsights/InfluxDBConnector/pubmanager"
	subManager "IEdgeInsights/InfluxDBConnector/submanager"
	"strconv"

	"github.com/golang/glog"
)

const (
	subServPort    = "61971"
	subServHost    = "localhost"
	influxCertPath = "/tmp/influxdb/ssl/influxdb_server_certificate.pem"
	influxKeyPath  = "/tmp/influxdb/ssl/influxdb_server_key.pem"
	influxCaPath   = "/tmp/influxdb/ssl/ca_certificate.pem"
	maxTopics      = 50
	maxSubTopics   = 50
)


// InfluxObj is an object for InfluxDB Manager
var InfluxObj dbManager.InfluxDBManager

var pubMgr pubManager.PubManager
var credConfig common.DbCredential
var runtimeInfo common.AppConfig

//Function to read the DB credential and container runtime info from the config file
func readConfig() {
	var errConfig error
	var errRuntimeInfo error
	credConfig, errConfig = configManager.ReadInfluxConfig()
	if errConfig != nil {
		glog.Error("Error in reading the DB credentials : %v" + errConfig.Error())
		os.Exit(-1)
	}

	runtimeInfo, errRuntimeInfo = configManager.ReadContainerInfo()
	if errRuntimeInfo != nil {
		glog.Error("Error in reading the Runtime Info : %v" + errRuntimeInfo.Error())
		os.Exit(-1)
	}
}

//StartDb Function to start Influx Database
//Initialize the Influx database with the configurations
func StartDb() {
	InfluxObj.DbInfo = credConfig
	InfluxObj.CnInfo = runtimeInfo
	err := InfluxObj.Init()
	if err != nil {
		glog.Errorf("StartDb: Failed to initialize InfluxDB : %v", err)
		os.Exit(-1)
	}

	err = InfluxObj.CreateDataBase(InfluxObj.DbInfo.Database, InfluxObj.DbInfo.Retention)
	if err != nil {
		glog.Errorf("StartDb: Failed to create database : %v", err)
		os.Exit(-1)
	}
}

// StartPublisher function to register the publisher and subscribe to influxdb
// ZeroMQ interface

func StartPublisher() {
	InfluxObj.CnInfo = runtimeInfo
	configMgr, err := eiscfgmgr.ConfigManager()

	if err != nil {
		glog.Fatalf("Config Manager initialization failed...")
	}
	numOfPublishers, err := configMgr.GetNumPublishers()
	if(err != nil) {
		glog.Errorf("Error occured with error:%v", err)
		return
	}
	pubMgr.Init()
	pubMgr.RegFilter(&InfluxObj)
	if numOfPublishers > maxTopics {
		glog.Infof("Max Topics Exceeded %d", numOfPublishers)
		return
	}

	for PubIndex := 0 ; PubIndex < numOfPublishers; PubIndex++ {		
		pubCtx, err := configMgr.GetPublisherByIndex(PubIndex)
		if(err != nil) {
			glog.Errorf("Error occured with error:%v", err)
			return
		}
		topic := pubCtx.GetTopics()[0]
		pubMgr.RegPublisherList(topic)
		glog.Infof("Publisher topic is : %s", topic)
		config, err := pubCtx.GetMsgbusConfig()
		if err != nil {
				glog.Error("Failed to get message bus config :%v", err)
				return
		}


		if config != nil {
			pubMgr.RegClientList(topic)
			pubMgr.CreateClient(topic, config)
		}

	}

	pubMgr.StartAllPublishers()
	var SubObj common.SubScriptionInfo
	SubObj.DbName = InfluxObj.DbInfo.Database
	SubObj.Host = subServHost
	SubObj.Port = subServPort
	SubObj.Worker = int(runtimeInfo.PubWorker)
	err = InfluxObj.Subscribe(SubObj, &pubMgr)
	if err != nil {
		glog.Errorf("StartPublisher: Failed to subscribe InfluxDB : %v", err)
		os.Exit(-1)
	}

}

//StartSubscriber Function to start the subscriber and insert data to influxdb
func StartSubscriber() {
	InfluxObj.CnInfo = runtimeInfo
	var subMgr subManager.SubManager
	var influxWrite dbManager.InfluxWriter
	var err error
	configMgr, err := eiscfgmgr.ConfigManager()
	if err != nil {
		glog.Fatalf("Config Manager initialization failed...")
	}
	numOfSubscribers, err := configMgr.GetNumSubscribers()
	if(err != nil) {
		glog.Errorf("Error occured with error:%v", err)
		return
	}
	influxWrite.DbInfo = credConfig
	influxWrite.CnInfo = runtimeInfo
	influxdbConnectorConfig, err := configManager.ReadInfluxDBConnectorConfig()
	if err != nil {
		glog.Error("Error in creating Ignore list")
	}
	influxWrite.IgnoreList = influxdbConnectorConfig["ignoreList"]
	influxWrite.TagList = influxdbConnectorConfig["tagsList"]

	subMgr.Init()
	if numOfSubscribers > maxSubTopics {
		glog.Infof("Max SubTopics Exceeded %d", numOfSubscribers)
		return
	}

	for SubIndex := 0; SubIndex < numOfSubscribers; SubIndex++ {
		
		subCtx, err := configMgr.GetSubscriberByIndex(SubIndex)
		if(err != nil) {
			glog.Errorf("Error occured with error:%v", err)
			return
		}

		topic := subCtx.GetTopics()[0] 
		glog.Infof("Subscriber topic is : %v", topic)

		subMgr.RegSubscriberList(topic)
		config, err := subCtx.GetMsgbusConfig()
		if err != nil {
				glog.Error("Failed to get message bus config :%v", err)
				return
		}

		if config != nil {
			subMgr.RegClientList(topic)
			subMgr.CreateClient(topic, config)
		}
	}

	subMgr.StartAllSubscribers()
	subMgr.ReceiveFromAll(&influxWrite, int(InfluxObj.CnInfo.SubWorker))
}

//Function to start the query server
func startReqReply() {

	InfluxObj.CnInfo = runtimeInfo
	configMgr, err := eiscfgmgr.ConfigManager()
	if err != nil {
		glog.Fatalf("Config Manager initialization failed...")
	}
	keyword, err := configMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}
	glog.Infof("Query service is : %s", keyword)
	serverCtx, err := configMgr.GetServerByIndex(0)
	if(err != nil) {
		glog.Errorf("Error occured with error:%v", err)
		return
	}
	config, err := serverCtx.GetMsgbusConfig()
	if(err != nil) {
		glog.Errorf("Error occured with error:%v", err)
		return
	}
	client, err := eismsgbus.NewMsgbusClient(config)
	if err != nil {
		glog.Errorf("-- Error initializing message bus context: %v\n", err)
		return
	}
	service, err := client.NewService(keyword)
	if err != nil {
		glog.Errorf("-- Error initializing service: %v\n", err)
		return
	}

	var influxQuery dbManager.InfluxQuery
	influxQuery.DbInfo = credConfig
	influxQuery.CnInfo = runtimeInfo
	influxdbQueryconfig, err := configManager.ReadInfluxDBQueryConfig()
	if err != nil {
		glog.Error("Error in creating query list")
		os.Exit(-1)
	}
	influxQuery.QueryListcon = influxdbQueryconfig

	influxQuery.Init()
	flag := true

	for flag {
		msg, err := service.ReceiveRequest(-1)
		if err != nil {
			glog.Errorf("-- Error receiving request: %v\n", err)
			return
		}
		glog.Infof("Command received: %s", msg)
		response, _ := influxQuery.QueryInflux(msg)
		service.Response(response.Data)
	}

}

//Function to stop the publishers
func cleanup() {
	pubMgr.StopAllClient()
	pubMgr.StopAllPublisher()
}

func main() {
	flag.Parse()
	profiling, _ := strconv.ParseBool(os.Getenv("PROFILING_MODE"))
	common.Profiling = profiling
	
	// Initializing Etcd to set env variables
	configMgr, err := eiscfgmgr.ConfigManager()
	if err != nil {
		glog.Fatalf("Config Manager initialization failed...")
	}
	devMode, err := configMgr.IsDevMode()
	if(err != nil) {
		glog.Fatalf("Error occured with error:%v", err)
	}
	if devMode != true {
		_ = configManager.ReadCertKey("server_cert", influxCertPath)
		_ = configManager.ReadCertKey("server_key", influxKeyPath)
		_ = configManager.ReadCertKey("ca_cert", influxCaPath)
	}
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", os.Getenv("GO_LOG_LEVEL"))
	flag.Set("v", os.Getenv("GO_VERBOSE"))
	done := make(chan bool)
	readConfig()
	StartDb()
	StartPublisher()
	StartSubscriber()
	go startReqReply()
	<-done
	cleanup()
}
