/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package pubManager

import (
	eismsgbus "EISMessageBus/eismsgbus"
	common "IEdgeInsights/InfluxDBConnector/common"

	"github.com/golang/glog"
)

//PubManager structure
type PubManager struct {
	// Will keep the map of endpoint name to the
	// client object
	clients map[string]*eismsgbus.MsgbusClient

	// Will keep the map of Topic Name/measurement Name
	// to the publisher object
	publishers map[string]*eismsgbus.Publisher

	// Info of registered Publishers
	pubConfigList []common.PubEndPoint

	//Info of registered Clients
	clientConfigList []common.Clients

	//This is for filtering the data
	filter common.Filter
}

//Init will initailize the maps
func (pubMgr *PubManager) Init() {
	pubMgr.clients = make(map[string]*eismsgbus.MsgbusClient)
	pubMgr.publishers = make(map[string]*eismsgbus.Publisher)
}

// RegPublisherList function will register the publishers and maintain
// pubEndPoint
func (pubMgr *PubManager) RegPublisherList(pubName string) error {

	pubMgr.pubConfigList = append(pubMgr.pubConfigList, common.PubEndPoint{pubName})

	return nil
}

// RegFilter function will register the filter
func (pubMgr *PubManager) RegFilter(fltr common.Filter) {

	pubMgr.filter = fltr
}

// RegClientList will register the clients and maintain
// Clients
func (pubMgr *PubManager) RegClientList(clientName string) error {

	pubMgr.clientConfigList = append(pubMgr.clientConfigList, common.Clients{clientName})
	return nil
}

// CreateClient will create the clients
func (pubMgr *PubManager) CreateClient(key string, config map[string]interface{}) error {

	var err error
	pubMgr.clients[key], err = eismsgbus.NewMsgbusClient(config)
	if err != nil {
		glog.Errorf("-- Error creating context: %v\n", err)
	}
	return nil
}

// StartAllPublishers function will start all the registered endpoints
// if not started already
func (pubMgr *PubManager) StartAllPublishers() error {

	var err error
	for _, pConfig := range pubMgr.pubConfigList {
		msgbusclient, ok := pubMgr.clients[pConfig.Name]
		if ok {
			pubMgr.publishers[pConfig.Name], err = msgbusclient.NewPublisher(pConfig.Name)
			if err != nil {
				glog.Errorf("-- Error creating publisher: %v\n", err)
			}

		}
	}

	return nil
}

func (pubMgr *PubManager) Write(data []byte) {

	attribute, err := pubMgr.filter.GetAttribute(data)
	if err != nil {
		glog.Errorf("server not responding %s", err.Error())
		return
	}
	pub, ok := pubMgr.publishers[attribute]
	if ok {
		msg := map[string]interface{}{"data": string(data)}
		glog.Infof("Published message: %v", msg)
		pub.Publish(msg)
	}
}

// StopAllPublisher function will stop all the registered publishers
func (pubMgr *PubManager) StopAllPublisher() {
	for _, pub := range pubMgr.publishers {
		pub.Close()
	}
}

// StopAllClient function will stop all the registered clients
func (pubMgr *PubManager) StopAllClient() {
	for _, client := range pubMgr.clients {
		client.Close()
	}
}
