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
	"strings"

	eismsgbus "EISMessageBus/eismsgbus"
	common "IEdgeInsights/InfluxDBConnector/common"
	configManager "IEdgeInsights/InfluxDBConnector/configManager"
	dbManager "IEdgeInsights/InfluxDBConnector/dbManager"
	pubManager "IEdgeInsights/InfluxDBConnector/pubManager"
	subManager "IEdgeInsights/InfluxDBConnector/subManager"
	configmgr "IEdgeInsights/libs/ConfigManager"
	util "IEdgeInsights/util"
	msgbusutil "IEdgeInsights/util/msgbusutil"
	"strconv"

	"github.com/golang/glog"
)

const (
	subServPort    = "61971"
	subServHost    = "localhost"
	influxCertPath = "/etc/ssl/influxdb/influxdb_server_certificate.pem"
	influxKeyPath  = "/etc/ssl/influxdb/influxdb_server_key.pem"
	influxCaPath   = "/etc/ssl/ca/ca_certificate.pem"
)

var cfgMgrConfig = map[string]string{
	"certFile":  "",
	"keyFile":   "",
	"trustFile": "",
}

// InfluxObj is an object for InfluxDB Manager
var InfluxObj dbManager.InfluxDBManager

var pubMgr pubManager.PubManager
var credConfig common.DbCredential
var runtimeInfo common.AppConfig

//Function to read the DB credential and container runtime info from the config file
func readConfig() {
	var errConfig error
	var errRuntimeInfo error
	credConfig, errConfig = configManager.ReadInfluxConfig(cfgMgrConfig)
	if errConfig != nil {
		glog.Error("Error in reading the DB credentials : %v" + errConfig.Error())
		os.Exit(-1)
	}

	runtimeInfo, errRuntimeInfo = configManager.ReadContainerInfo(cfgMgrConfig)
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
	keywords := os.Getenv("PubTopics")
	keyword := strings.Split(keywords, ",")
	pubMgr.Init()
	pubMgr.RegFilter(&InfluxObj)

	for _, key := range keyword {
		glog.Infof("Publisher topic is : %s", key)
		pubMgr.RegPublisherList(key)
		cConfigList := msgbusutil.GetMessageBusConfig(key, "pub", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

		if cConfigList != nil {
			pubMgr.RegClientList(key)
			pubMgr.CreateClient(key, cConfigList)
		}

	}

	pubMgr.StartAllPublishers()
	var SubObj common.SubScriptionInfo
	SubObj.DbName = InfluxObj.DbInfo.Database
	SubObj.Host = subServHost
	SubObj.Port = subServPort
	SubObj.Worker = int(runtimeInfo.PubWorker)
	// Subscribe to the influxdb database
	err := InfluxObj.Subscribe(SubObj, &pubMgr)
	if err != nil {
		glog.Errorf("StartPublisher: Failed to subscribe InfluxDB : %v", err)
		os.Exit(-1)
	}

}

//StartSubscriber Function to start the subscriber and insert data to influxdb
func StartSubscriber() {
	var SubKeyword []string
	InfluxObj.CnInfo = runtimeInfo
	keywords := os.Getenv("SubTopics")
	if len(keywords) == 0 {
		return
	}
	keyword := strings.Split(keywords, ",")

	var subMgr subManager.SubManager
	var influxWrite dbManager.InfluxWriter
	influxWrite.DbInfo = credConfig
	influxWrite.CnInfo = runtimeInfo
	subMgr.Init()

	for _, key := range keyword {
		SubKeyword = strings.Split(key, "/")
		glog.Infof("Subscriber topic is : %v", SubKeyword[1])

		subMgr.RegSubscriberList(SubKeyword[1])
		cConfigList := msgbusutil.GetMessageBusConfig(key, "sub", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

		if cConfigList != nil {
			subMgr.RegClientList(SubKeyword[1])
			subMgr.CreateClient(SubKeyword[1], cConfigList)
		}
	}

	subMgr.StartAllSubscribers()
	subMgr.ReceiveFromAll(&influxWrite, int(InfluxObj.CnInfo.SubWorker))
}

//Function to start the query server
func startReqReply() {

	InfluxObj.CnInfo = runtimeInfo
	keyword := os.Getenv("AppName")

	glog.Infof("Query service is : %s", keyword)

	cConfigList := msgbusutil.GetMessageBusConfig(keyword, "server", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

	client, err := eismsgbus.NewMsgbusClient(cConfigList)
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

	appName := os.Getenv("AppName")
	cfgMgrConfig = util.GetCryptoMap(appName)

	_ = configManager.ReadCertKey("server_cert", influxCertPath, cfgMgrConfig)
	_ = configManager.ReadCertKey("server_key", influxKeyPath, cfgMgrConfig)
	_ = configManager.ReadCertKey("ca_cert", influxCaPath, cfgMgrConfig)

	// Initializing Etcd to set env variables
	_ = configmgr.Init("etcd", cfgMgrConfig)
	flag.Lookup("alsologtostderr").Value.Set("true")
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
