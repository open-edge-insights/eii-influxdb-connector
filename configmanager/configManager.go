/*
Copyright (c) 2021 Intel Corporation

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package configmanager

import (
	util "IEdgeInsights/common/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	eiicfgmgr "ConfigMgr/eiiconfigmgr"
	common "IEdgeInsights/InfluxDBConnector/common"

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

// configmanager structure
type ConfigManager struct {
	ConfigMgr *eiicfgmgr.ConfigMgr
}

//Init will initailize the maps
func (CfgMgr *ConfigManager) Init() {
	CfgMgr.ConfigMgr, _ = eiicfgmgr.ConfigManager()
	if CfgMgr.ConfigMgr == nil {
		glog.Fatalf("Config Manager initialization failed...")
	}
}
// ReadInfluxConfig will read the influxdb configuration
// from the json file
func (CfgMgr *ConfigManager) ReadInfluxConfig() (common.DbCredential, error) {
	var influx InfluxConfig
	var influxCred common.DbCredential

	appName, err := CfgMgr.ConfigMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}

	data, err := CfgMgr.ConfigMgr.GetAppConfig()
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return influxCred, err
	}
	value, _ := json.Marshal(data)
	// Reading schema json
	schema, err := ioutil.ReadFile("./schema.json")
	if err != nil {
		glog.Errorf("Schema file not found")
		return influxCred, err
	}

	// Validating config json
	if util.ValidateJSON(string(schema), string(value)) != true {
		return influxCred, err
	}
	err = json.Unmarshal([]byte(value), &influx)
	if err != nil {
		glog.Errorf("json error: %s", err.Error())
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
// for the subworkers, pubworkers and DEV mode info
func (CfgMgr *ConfigManager) ReadContainerInfo() (common.AppConfig, error) {

	var cInfo common.AppConfig
	devMode, err := CfgMgr.ConfigMgr.IsDevMode()
	cInfo.DevMode = devMode
	if err != nil {
		glog.Errorf("Fail to read DEV_MODE from etcd: %v", err)
		return cInfo, err
	}


	appName, err := CfgMgr.ConfigMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}

	data, err := CfgMgr.ConfigMgr.GetAppConfig()
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return cInfo, err
	}

	cInfo.PubWorker, err = strconv.ParseInt(data["pub_workers"].(string), 10, 0)
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return cInfo, err
	}
	cInfo.SubWorker, err = strconv.ParseInt(data["sub_workers"].(string), 10, 0)
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return cInfo, err
	}

	return cInfo, nil
}

// ReadCertKey will read the certificate from etcd
// and write to path passed as argument
func (CfgMgr *ConfigManager) ReadCertKey(keyName string, filePath string) error {

	appName, err := CfgMgr.ConfigMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}

	data, err := CfgMgr.ConfigMgr.GetAppConfig()
	if err != nil {
		glog.Errorf("Not able to read value from etcd for / %v / %v", appName, keyName)
		return err
	}

	value := fmt.Sprintf("%v", data[keyName])
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return err
	}

	if filePath != "" {
		err = ioutil.WriteFile(filePath, []byte(value), 0644)
		if err != nil {
			glog.Errorf("Error creating %v", filePath)
			return err
		}
	} else {
		glog.Errorf("File Path is Empty")
	}

	return nil
}

// ReadInfluxDBConnectorConfig will read the file
// and create an Ignore list
func (CfgMgr *ConfigManager) ReadInfluxDBConnectorConfig() (map[string][]string, error) {
	influxdbConnCon := make(map[string][]string)

	appName, err := CfgMgr.ConfigMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}

	data, err := CfgMgr.ConfigMgr.GetAppConfig()
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return influxdbConnCon, err
	}

	for tags, value := range data {

		if tags == "ignore_keys" {
			if value != nil {
				for _, keys := range value.([]interface{}) {
					influxdbConnCon["ignoreList"] = append(influxdbConnCon["ignoreList"], fmt.Sprintf("%v", keys))
				}
			}
		}
		if tags == "tag_keys" {
			if value != nil {
				for _, keys := range value.([]interface{}) {
					influxdbConnCon["tagsList"] = append(influxdbConnCon["tagsList"], fmt.Sprintf("%v", keys))
				}
			}
		}
	}

	glog.Infof("Influxdbconnector configs are: %v", influxdbConnCon)
	return influxdbConnCon, nil
}


// ReadInfluxDBQueryConfig will read the file
// and create a Blacklist QueryList
func (CfgMgr *ConfigManager) ReadInfluxDBQueryConfig() (map[string][]string, error) {

	influxdbQuerycon := make(map[string][]string)

	appName, err := CfgMgr.ConfigMgr.GetAppName()
	if err != nil {
		glog.Fatalf("Not able to read appname from etcd")
	}

	data, err := CfgMgr.ConfigMgr.GetAppConfig()
	if err != nil {
		glog.Errorf("Not able to read value from etcd for /%v/config", appName)
		return influxdbQuerycon, err
	}

	for tags, value := range data {

		// parsing the config to fetch required value
		if tags == "blacklist_query" {
			if value != nil {
				for _, keys := range value.([]interface{}) {
					influxdbQuerycon["BlacklistQueryList"] = append(influxdbQuerycon["BlacklistQueryList"], fmt.Sprintf("%v", keys))
				}
			}
		}
	}

	glog.Infof("Successfully read black listed item in query")
	return influxdbQuerycon, nil
}
