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

	configmgr "ConfigManager"
	eismsgbus "EISMessageBus/eismsgbus"
	envconfig "EnvConfig"
	common "IEdgeInsights/InfluxDBConnector/common"
	configManager "IEdgeInsights/InfluxDBConnector/configmanager"
	dbManager "IEdgeInsights/InfluxDBConnector/dbmanager"
	pubManager "IEdgeInsights/InfluxDBConnector/pubmanager"
	subManager "IEdgeInsights/InfluxDBConnector/submanager"
	util "IEdgeInsights/common/util"
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
func StartPublisher(pubTopics string) {

	InfluxObj.CnInfo = runtimeInfo
	keyword := strings.Split(pubTopics, ",")
	pubMgr.Init()
	pubMgr.RegFilter(&InfluxObj)
	if len(keyword) > maxTopics {
		glog.Infof("Max Topics Exceeded %d", len(keyword))
		return
	}

	for _, key := range keyword {
		glog.Infof("Publisher topic is : %s", key)
		pubMgr.RegPublisherList(key)
		cConfigList := envconfig.GetMessageBusConfig(key, "pub", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

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
func StartSubscriber(subTopics string) {
	var SubKeyword []string
	InfluxObj.CnInfo = runtimeInfo
	keyword := strings.Split(subTopics, ",")

	var subMgr subManager.SubManager
	var influxWrite dbManager.InfluxWriter
	var err error

	influxWrite.DbInfo = credConfig
	influxWrite.CnInfo = runtimeInfo
	influxdbConnectorConfig, err := configManager.ReadInfluxDBConnectorConfig(cfgMgrConfig)
	if err != nil {
		glog.Error("Error in creating Ignore list")
	}
	influxWrite.IgnoreList = influxdbConnectorConfig["ignoreList"]
	influxWrite.TagList = influxdbConnectorConfig["tagsList"]

	subMgr.Init()
	if len(keyword) > maxSubTopics {
		glog.Infof("Max SubTopics Exceeded %d", len(keyword))
		return
	}

	for _, key := range keyword {
		SubKeyword = strings.Split(key, "/")
		glog.Infof("Subscriber topic is : %v", SubKeyword[1])

		subMgr.RegSubscriberList(SubKeyword[1])
		cConfigList := envconfig.GetMessageBusConfig(key, "sub", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

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

	cConfigList := envconfig.GetMessageBusConfig(keyword, "server", InfluxObj.CnInfo.DevMode, cfgMgrConfig)

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
	cfgMgrCli := configmgr.Init("etcd", cfgMgrConfig)
	if cfgMgrCli == nil {
		glog.Fatalf("Config Manager initialization failed...")
	}

	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", os.Getenv("GO_LOG_LEVEL"))
	flag.Set("v", os.Getenv("GO_VERBOSE"))
	done := make(chan bool)
	readConfig()
	StartDb()
	pubTopics := os.Getenv("PubTopics")
	if pubTopics != "" {
		StartPublisher(pubTopics)
	} else {
		glog.Infof("Not starting Publisher since PubTopics env is not set")
	}
	subTopics := os.Getenv("SubTopics")
	if subTopics != "" {
		StartSubscriber(subTopics)
	} else {
		glog.Infof("Not starting Subscriber since SubTopics env is not set")
	}
	go startReqReply()
	<-done
	cleanup()
}
