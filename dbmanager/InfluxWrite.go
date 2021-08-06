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

package dbmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	common "IEdgeInsights/InfluxDBConnector/common"
	inflxUtil "IEdgeInsights/common/util/influxdb"
	"github.com/golang/glog"
	"github.com/influxdata/influxdb/client/v2"
)

// InfluxWriter structure
type InfluxWriter struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	CnInfo      common.AppConfig
	DbInfo      common.DbCredential
	IgnoreList  []string
	TagList     []string
}

func (ir *InfluxWriter) parseData(msg []byte, topic string) *InfluxWriter {
	tags := make(map[string]string)
	field := make(map[string]interface{})
	data := make(map[string]interface{})
	var tempir InfluxWriter

	err := json.Unmarshal(msg, &data)

	if err != nil {
		glog.Errorf("Not able to Parse data %s", err.Error())
		return nil
	}

	if common.Profiling == true {
		data["tsIdbconnProcEntry"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	for key, value := range data {
		for _, tagkey := range ir.TagList {
			if key == tagkey {
				tags[key] = fmt.Sprintf("%v", value)
				delete(data, key)
			}
		}
	}

	flatjson, err := ir.getflatten(data, "")
	if err != nil {
		glog.Errorf("Not able to flatten json %s for:%v", err.Error(), data)
		return nil
	}

	glog.Infof("Data after flattening: %v", flatjson)

	for key, value := range flatjson {
		if reflect.ValueOf(value).Type().Kind() == reflect.Float64 {
			field[key] = value
		} else if reflect.ValueOf(value).Type().Kind() == reflect.String {
			field[key] = value
		} else if reflect.ValueOf(value).Type().Kind() == reflect.Bool {
			field[key] = value
		} else if reflect.ValueOf(value).Type().Kind() == reflect.Int {
			field[key] = value
		}
	}

	tempir.Measurement = topic
	tempir.Tags = tags
	tempir.Fields = field

	return &tempir
}

func (ir *InfluxWriter) getflatten(nested map[string]interface{}, prefix string) (map[string]interface{}, error) {
	flatmap := make(map[string]interface{})

	err := flatten(true, flatmap, nested, prefix, ir.IgnoreList)
	if err != nil {
		return nil, err
	}

	return flatmap, nil
}

func flatten(top bool, flatMap map[string]interface{}, nested interface{}, prefix string, ignorelist []string) error {
	var flag int

	assign := func(newKey string, v interface{}, ignoretag bool) error {
		if ignoretag {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				v, err := json.Marshal(&v)
				if err != nil {
					glog.Error("Not able to Marshal data for key:%s=%v", newKey, v)
					return err
				}
				flatMap[newKey] = string(v)
			default:
				flatMap[newKey] = v
			}

		} else {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				if err := flatten(false, flatMap, v, newKey, ignorelist); err != nil {
					glog.Error("Not able to flatten data for key:%s=%v", newKey, v)
					return err
				}
			default:
				flatMap[newKey] = v
			}
		}
		return nil
	}

	switch nested.(type) {
	case map[string]interface{}:
		for k, v := range nested.(map[string]interface{}) {

			ok := matchkey(ignorelist, k)

			if ok && prefix == "" {
				flag = 1
			} else if ok && prefix != "" {
				flag = 0
			} else {
				flag = -1
			}

			if flag == 1 {
				err := assign(k, v, true)
				if err != nil {
					return err
				}
			} else if flag == 0 {
				newKey := createkey(top, prefix, k)
				err := assign(newKey, v, true)
				if err != nil {
					return err
				}
			} else {
				newKey := createkey(top, prefix, k)
				err := assign(newKey, v, false)
				if err != nil {
					return err
				}
			}
		}
	case []interface{}:
		for i, v := range nested.([]interface{}) {
			switch v.(type) {
			case map[string]interface{}:
				for tag, value := range v.(map[string]interface{}) {
					ok := matchkey(ignorelist, tag)
					if ok {
						subkey := strconv.Itoa(i) + "." + tag
						newKey := createkey(top, prefix, subkey)
						err := assign(newKey, value, true)
						if err != nil {
							return err
						}
					} else {
						newKey := createkey(top, prefix, strconv.Itoa(i))
						err := assign(newKey, v, false)
						if err != nil {
							return err
						}
					}
				}
			default:
				newKey := createkey(top, prefix, strconv.Itoa(i))
				err := assign(newKey, v, false)
				if err != nil {
					return err
				}
			}

		}
	default:
		return errors.New("Not a valid input: map or slice")
	}

	return nil
}

func createkey(top bool, prefix, subkey string) string {
	key := prefix

	if top {
		key += subkey
	} else {
		key += "." + subkey
	}

	return key
}

func matchkey(ignorelist []string, value string) bool {

	for _, val := range ignorelist {
		if val == value {
			return true
		}
	}

	return false
}

func (ir *InfluxWriter) insertData(data *InfluxWriter) {

	if common.Profiling == true {
		data.Fields["tsIdbconnHTTPEntry"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	clientadmin, err := inflxUtil.CreateHTTPClient(ir.DbInfo.Host, ir.DbInfo.Port, ir.DbInfo.Username, ir.DbInfo.Password, ir.CnInfo.DevMode)

	defer clientadmin.Close()

	if common.Profiling == true {
		data.Fields["tsIdbconnHTTPClientReady"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  ir.DbInfo.Database,
		Precision: "ns",
	})

	if err != nil {
		glog.Errorf("Error in creating batch point %s", err.Error())
	}

	pt, err := client.NewPoint(data.Measurement, data.Tags, data.Fields, time.Now())
	if err != nil {
		glog.Errorf("point error %s", err.Error())
		os.Exit(-1)
	}

	bp.AddPoint(pt)

	if common.Profiling == true {
		data.Fields["tsIdbconnHTTPBatchpointReady"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	if err := clientadmin.Write(bp); err != nil {
		glog.Errorf("Write Error %s", err.Error())
	}

	if common.Profiling == true {
		tsIdbconnExit := (time.Now().UnixNano() / 1e6)

		tsIdbconnHTTPEntry, _ := strconv.ParseInt(data.Fields["tsIdbconnHTTPEntry"].(string), 10, 64)
		tsIdbconnProcEntry, _ := strconv.ParseInt(data.Fields["tsIdbconnProcEntry"].(string), 10, 64)
		tsIdbconnEntry, _ := strconv.ParseInt(data.Fields["tsIdbconnEntry"].(string), 10, 64)
		tsIdbconnHTTPClientReady, _ := strconv.ParseInt(data.Fields["tsIdbconnHTTPClientReady"].(string), 10, 64)
		tsIdbconnHTTPBatchpointReady, _ := strconv.ParseInt(data.Fields["tsIdbconnHTTPBatchpointReady"].(string), 10, 64)

		tmIdbconnHTTPProc := tsIdbconnExit - tsIdbconnHTTPEntry
		tmIdbconnJSONProc := tsIdbconnProcEntry - tsIdbconnHTTPEntry
		tmLatencyAtInfluxdbconnector := tsIdbconnExit - tsIdbconnEntry
		tmHTTPClientCreation := tsIdbconnHTTPClientReady - tsIdbconnHTTPEntry
		tmBatchpointReady := tsIdbconnHTTPBatchpointReady - tsIdbconnHTTPClientReady

		glog.Infof("======Start=====")
		glog.Infof("Lattency:%v", tmLatencyAtInfluxdbconnector)
		glog.Infof("ts_idbconn_http_proc:%v", tmIdbconnHTTPProc)
		glog.Infof("ts_idbconn_json_proc:%v", tmIdbconnJSONProc)
		glog.Infof("tmHTTPClientCreation:%v", tmHTTPClientCreation)
		glog.Infof("tmBatchpointReady:%v", tmBatchpointReady)

		glog.Infof("======End=====")
	}

}

func (ir *InfluxWriter) Write(data []byte, topic string) {
	InfluxRecord := ir.parseData(data, topic)
	ir.insertData(InfluxRecord)
}
