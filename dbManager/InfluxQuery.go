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
	"errors"

	types "EISMessageBus/pkg/types"
	common "IEdgeInsights/InfluxDBConnector/common"
	inflxUtil "IEdgeInsights/libs/common/influxdb"

	"github.com/golang/glog"
	"github.com/influxdata/influxdb/client/v2"
)

// InfluxQuery structure
type InfluxQuery struct {
	CnInfo common.AppConfig
	DbInfo common.DbCredential
}

// QueryInflux will execute the select command and
// return the response
func (iq *InfluxQuery) QueryInflux(msg *types.MsgEnvelope) (*types.MsgEnvelope, error) {
	clientadmin, err := inflxUtil.CreateHTTPClient(iq.DbInfo.Host, iq.DbInfo.Port, iq.DbInfo.Username, iq.DbInfo.Password, iq.CnInfo.DevMode)

	if err != nil {
		glog.Errorf("client error %s", err)
	}
	Command, ok := msg.Data["command"].(string)
	if ok {
		q := client.Query{
			Command:   Command,
			Database:  iq.DbInfo.Database,
			Precision: "s",
		}

		if response, err := clientadmin.Query(q); err == nil && response.Error() == nil {
			
			if len(response.Results[0].Series) > 0 {
				output := response.Results[0].Series[0]
				glog.V(1).Infof("%v", output)
				Output, err := json.Marshal(output)
				response := types.NewMsgEnvelope(map[string]interface{}{"Data": string(Output)}, nil)
				return response, err
			}
			val := types.NewMsgEnvelope(map[string]interface{}{"Data": ""}, nil)
			err = errors.New("Response is nil")
			return val, err
		} else {
			glog.V(1).Infof("%v", response)
			glog.V(1).Infof("%v", response.Error())
		} 

	}
	val := types.NewMsgEnvelope(map[string]interface{}{"Data": ""}, nil)
	err = errors.New("Please send proper select query")
	return val, err
}
