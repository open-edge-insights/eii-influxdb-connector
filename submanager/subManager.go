/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package submanager

import (
	eismsgbus "EISMessageBus/eismsgbus"
	common "IEdgeInsights/InfluxDBConnector/common"
	"encoding/json"
	"strconv"
	"time"

	"github.com/golang/glog"
)

//SubManager structure
type SubManager struct {
	// Will keep the map of Topic Name/measurement Name
	// to the publisher object
	subscribers map[string]*eismsgbus.Subscriber

	clients map[string]*eismsgbus.MsgbusClient

	// Info of registered Publishers
	subConfigList []common.SubEndPoint

	// Info of registered clients
	clientConfigList []common.Clients
}

//Init will initailize the maps
func (subMgr *SubManager) Init() {
	subMgr.clients = make(map[string]*eismsgbus.MsgbusClient)
	subMgr.subscribers = make(map[string]*eismsgbus.Subscriber)
}

// RegSubscriberList function will register the publishers and maintain
// pubEndPoint
func (subMgr *SubManager) RegSubscriberList(subName string) error {

	subMgr.subConfigList = append(subMgr.subConfigList, common.SubEndPoint{subName})

	return nil
}

// RegClientList will register the clients and maintain
// pubEndPoint
func (subMgr *SubManager) RegClientList(clientName string) error {

	subMgr.clientConfigList = append(subMgr.clientConfigList, common.Clients{clientName})
	return nil
}

// CreateClient will create the clients
func (subMgr *SubManager) CreateClient(key string, config map[string]interface{}) error {

	var err error
	subMgr.clients[key], err = eismsgbus.NewMsgbusClient(config)
	if err != nil {
		glog.Errorf("-- Error creating context: %v\n", err)
	}

	return nil
}

// StartAllSubscribers function will start all teh registered endpoints
// if not started already
func (subMgr *SubManager) StartAllSubscribers() error {

	for _, pConfig := range subMgr.subConfigList {
		msgbusclient, ok := subMgr.clients[pConfig.Measurement]
		if ok {
			tempSub, err := msgbusclient.NewSubscriber(pConfig.Measurement)
			if err != nil {
				glog.Errorf("-- Error creating Subscribers: %v\n", err)
			} else {
				subMgr.subscribers[pConfig.Measurement] = tempSub
			}
		}
	}

	return nil
}

// ReceiveFromAll function will receive data from all the subscriber
// end points
func (subMgr *SubManager) ReceiveFromAll(out common.InsertInterface, worker int) {
	glog.Infof("Subscriber available is: %v", subMgr.subscribers)
	for topic, sub := range subMgr.subscribers {
		glog.Infof("Subscriber topic is: %s", topic)
		for workerID := 0; workerID < worker; workerID++ {
			go processMsg(sub, out, workerID)
		}
	}
}

func processMsg(sub *eismsgbus.Subscriber, out common.InsertInterface, workerID int) {
	for {
		msg := <-sub.MessageChannel
		// parse it get the InfluxRow object ir.

		if common.Profiling == true {
			msg.Data["tsIdbconnEntry"] = strconv.FormatInt((time.Now().UnixNano() / 1e6), 10)
		}

		bytemsg, err := json.Marshal(msg.Data)
		if err != nil {
			glog.Errorf("error: %s", err)
		}

		glog.Infof("Subscribe data received from topic: %s in subroutine %v", msg.Name, workerID)
		out.Write(bytemsg, msg.Name)
	}
}

// StopAllSubscribers function will stop all the registered subscriber
func (subMgr *SubManager) StopAllSubscribers() {
	for _, sub := range subMgr.subscribers {
		sub.Close()
	}
}

// StopAllClient function will stop all the registered client
func (subMgr *SubManager) StopAllClient() {
	for _, client := range subMgr.clients {
		client.Close()
	}
}
