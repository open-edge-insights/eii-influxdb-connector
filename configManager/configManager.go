/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package configManager

import (
	"encoding/json"
	"os"
	"strconv"

	common "IEdgeInsights/InfluxDBConnector/common"
        util "IEdgeInsights/libs/common/go"
	configmgr "IEdgeInsights/libs/ConfigManager"

	"github.com/golang/glog"
)

//InfluxConfig structure
type InfluxConfig struct {
	Influxdb struct {
		Retention string `json:"Retention"`
		Username  string `json:"Username"`
		Password  string `json:"Password"`
		Dbname    string `json:"Dbname"`
		Ssl       string `json:"Ssl"`
		VerifySsl string `json:"VerifySsl"`
		Port      string `json:"Port"`
	} `json:"influxdb"`
}

// ReadClientConfigFromFile will read the publisher/subscriber client config
// from the json config file
func ReadClientConfig(topic string, topicType string) (map[string]interface{}) {
        //client := make(map[string]interface{})
	if topicType == "service" {
		endPoint := os.Getenv("Server")
		glog.Infof("Server config is :%s", endPoint)
                client, _ := util.GetEndPointMap(topic,endPoint) 
                return client
	} else {
	        client := util.GetTopicConfig(topic, topicType)
                return client
        }
}

// ReadInfluxConfig will read the influxdb configuration
// from the json file
func ReadInfluxConfig() (common.DbCredential, error) {
	var influx InfluxConfig
	var influxCred common.DbCredential
	config := map[string]string{
		      "CertFile": "",
		      "KeyFile": "",
		      "TrustFile": "",
	      }

	mgr := configmgr.Init("etcd", config)
        appName := os.Getenv("AppName")
	value, err := mgr.GetConfig("/"+appName+"/config")
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /InfluxDBConnector/influxdb_config")
		return influxCred, err
	}

	err = json.Unmarshal([]byte(value), &influx)
	if err != nil {
		glog.Errorf("json error:", err.Error())
		return influxCred, err
	}

	influxCred.Username = influx.Influxdb.Username
	influxCred.Password = influx.Influxdb.Password
	influxCred.Database = influx.Influxdb.Dbname
	influxCred.Retention = influx.Influxdb.Retention
	influxCred.Port = influx.Influxdb.Port
	influxCred.Ssl = influx.Influxdb.Ssl
	influxCred.Verifyssl = influx.Influxdb.VerifySsl
	influxCred.Host = "localhost"

	return influxCred, nil
}

// ReadContainerInfo will read the environment variable
// for the TPM and DEV mode info
func ReadContainerInfo() (common.ContainerConfig, error) {

	var cInfo common.ContainerConfig
	var err error
	devMode := os.Getenv("DEV_MODE")
	cInfo.DevMode, err = strconv.ParseBool(devMode)
	if err != nil {
		glog.Errorf("Fail to read DEV_MODE environment variable: %s", err)
		return cInfo, err
	}

	return cInfo, nil
}
