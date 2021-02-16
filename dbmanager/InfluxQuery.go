/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dbmanager

import (
	"encoding/json"
	"errors"
	"regexp"

	types "EIIMessageBus/pkg/types"
	common "IEdgeInsights/InfluxDBConnector/common"
	inflxUtil "IEdgeInsights/common/util/influxdb"

	"github.com/golang/glog"
	"github.com/influxdata/influxdb/client/v2"
	"strings"
)

// InfluxQuery structure
type InfluxQuery struct {
	CnInfo         common.AppConfig
	DbInfo         common.DbCredential
	queryWhitelistValidator *regexp.Regexp
	queryBlacklistValidator *regexp.Regexp
	QueryListcon map[string][]string
}

// QueryInflux will block the blacklist queries, execute the select command and
// return the response
func (iq *InfluxQuery) QueryInflux(msg *types.MsgEnvelope) (*types.MsgEnvelope, error) {
	var validQuery bool
	var invalidQuery bool
	clientadmin, err := inflxUtil.CreateHTTPClient(iq.DbInfo.Host, iq.DbInfo.Port, iq.DbInfo.Username, iq.DbInfo.Password, iq.CnInfo.DevMode)

	if err != nil {
		glog.Errorf("client error %s", err)
	}
	command, ok := msg.Data["command"].(string)
	cmdL := strings.ToLower(command)
	if len(iq.QueryListcon) == 0 {
		invalidQuery = false
	} else {
		invalidQuery = iq.queryBlacklistValidator.MatchString(cmdL)
	}
	if !invalidQuery {
		validQuery = iq.queryWhitelistValidator.MatchString(cmdL)
	} else {
		glog.Infof("Query is blacklisted")
	}

	if ok && validQuery {
		q := client.Query{
			Command:   command,
			Database:  iq.DbInfo.Database,
			Precision: "s",
		}

		if response, err := clientadmin.Query(q); err == nil {

			if response.Error() != nil {
				glog.V(1).Infof("Response received: %v", response)
				glog.V(1).Infof("Response Error received: %v", response.Error())
			} else {
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
			}
		}

	}
	val := types.NewMsgEnvelope(map[string]interface{}{"Data": ""}, nil)
	err = errors.New("Please send proper select query")
	return val, err
}



// Init function to check if select query is passed and forming regular expression for black list queries.
func (iq *InfluxQuery) Init() {
	var blacklist string

	for _,value := range iq.QueryListcon["BlacklistQueryList"] {
		value = strings.ToLower(value)
		// Regex is used for matching the query containing elements of Blacklist QueryList.Here '\s+' is used to match one or more whitespace charecter
                // and '.*' is used to match zero or more number of any characters. '^' signifies start of the line and '$' signifies end of the line.
		blacklistexp := "^(" + value + "\\s+.*)|(.*\\s+" + value + "\\s+.*)|(.*\\s+" + value + "$)|^(" + value + "$)|(.*;" + value + "\\s+.*)|(.*;" + value + "$)"
		if len(strings.TrimSpace(blacklist)) == 0 {
			blacklist += blacklistexp
		} else {
			blacklist += "|" + blacklistexp
		}
	}

	iq.queryBlacklistValidator = regexp.MustCompile("(" + blacklist + ")")
	iq.queryWhitelistValidator = regexp.MustCompile("^(select\\s+.*)")
}
