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
	"time"

	common "IEdgeInsights/InfluxDBConnector/common"
	inflxUtil "IEdgeInsights/libs/common/influxdb"

	"github.com/golang/glog"
	"github.com/influxdata/influxdb/client/v2"
)

// InfluxWriter structure
type InfluxWriter struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}

	CnInfo common.AppConfig
	DbInfo common.DbCredential
}

func (ir *InfluxWriter) parseData(msg []byte, topic string) {
	tags := make(map[string]string)
	field := make(map[string]interface{})
	data := make(map[string]interface{})

	err := json.Unmarshal(msg, &data)

	if err != nil {

		glog.Errorf("Not able to Parse data %s", err.Error())
		return
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

	ir.Measurement = topic
	ir.Tags = tags
	ir.Fields = field
}

func (ir *InfluxWriter) insertData() {
	clientadmin, err := inflxUtil.CreateHTTPClient(ir.DbInfo.Host, ir.DbInfo.Port, ir.DbInfo.Username, ir.DbInfo.Password, ir.CnInfo.DevMode)

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  ir.DbInfo.Database,
		Precision: "ns",
	})

	if err != nil {
		glog.Errorf("Error in creating batch point %s", err.Error())
	}

	pt, err := client.NewPoint(ir.Measurement, ir.Tags, ir.Fields, time.Now())
	if err != nil {
		glog.Errorf("point error %s", err.Error())
		os.Exit(-1)
	}

	bp.AddPoint(pt)

	if err := clientadmin.Write(bp); err != nil {
		glog.Errorf("Write Error %s", err.Error())
	}
}

func (ir *InfluxWriter) Write(data []byte, topic string) {
	ir.parseData(data, topic)
	ir.insertData()
}
