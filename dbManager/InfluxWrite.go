/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dbManager

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"time"

	common "IEdgeInsights/InfluxDBConnector/common"
	inflxUtil "IEdgeInsights/util/influxdb"

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
		data["ts_idbconn_proc_entry"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	for key, value := range data {
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

func (ir *InfluxWriter) insertData(data *InfluxWriter) {

	if common.Profiling == true {
		data.Fields["ts_idbconn_http_entry"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	clientadmin, err := inflxUtil.CreateHTTPClient(ir.DbInfo.Host, ir.DbInfo.Port, ir.DbInfo.Username, ir.DbInfo.Password, ir.CnInfo.DevMode)

	if common.Profiling == true {
		data.Fields["ts_idbconn_http_client_ready"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
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
		data.Fields["ts_idbconn_http_batchpoint_ready"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
	}

	if err := clientadmin.Write(bp); err != nil {
		glog.Errorf("Write Error %s", err.Error())
	}

	if common.Profiling == true {
		ts_idbconn_exit := (time.Now().UnixNano() / 1e6)

		ts_idbconn_http_entry, _ := strconv.ParseInt(data.Fields["ts_idbconn_http_entry"].(string), 10, 64)
		ts_idbconn_proc_entry, _ := strconv.ParseInt(data.Fields["ts_idbconn_proc_entry"].(string), 10, 64)
		ts_idbconn_entry, _ := strconv.ParseInt(data.Fields["ts_idbconn_entry"].(string), 10, 64)
		ts_idbconn_http_client_ready, _ := strconv.ParseInt(data.Fields["ts_idbconn_http_client_ready"].(string), 10, 64)
		ts_idbconn_http_batchpoint_ready, _ := strconv.ParseInt(data.Fields["ts_idbconn_http_batchpoint_ready"].(string), 10, 64)

		tm_idbconn_http_proc := ts_idbconn_exit - ts_idbconn_http_entry
		tm_idbconn_json_proc := ts_idbconn_proc_entry - ts_idbconn_http_entry
		tm_latency_at_influxdbconnector := ts_idbconn_exit - ts_idbconn_entry
		tm_http_client_creation := ts_idbconn_http_client_ready - ts_idbconn_http_entry
		tm_batchpoint_ready := ts_idbconn_http_batchpoint_ready - ts_idbconn_http_client_ready

		glog.Infof("======Start=====")
		glog.Infof("Lattency:%v", tm_latency_at_influxdbconnector)
		glog.Infof("ts_idbconn_http_proc:%v", tm_idbconn_http_proc)
		glog.Infof("ts_idbconn_json_proc:%v", tm_idbconn_json_proc)
		glog.Infof("tm_http_client_creation:%v", tm_http_client_creation)
		glog.Infof("tm_batchpoint_ready:%v", tm_batchpoint_ready)

		glog.Infof("======End=====")
	}

}

func (ir *InfluxWriter) Write(data []byte, topic string) {
	InfluxRecord := ir.parseData(data, topic)
	ir.insertData(InfluxRecord)
}
