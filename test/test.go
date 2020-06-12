/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package main

import (
	eismsgbus "EISMessageBus/eismsgbus"
	types "EISMessageBus/pkg/types"
	"flag"
	"fmt"
)

func testPublisher(config map[string]interface{}, topic string) {
	fmt.Println("-- Initializing message bus context")
	client, err := eismsgbus.NewMsgbusClient(config)
	if err != nil {
		fmt.Printf("-- Error initializing message bus context: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Printf("-- Creating publisher for topic %s\n", topic)
	publisher, err := client.NewPublisher(topic)
	if err != nil {
		fmt.Printf("-- Error creating publisher: %v\n", err)
		return
	}
	defer publisher.Close()

	msg := map[string]interface{}{"Measurement": "stream_results",
		"ImageStore": "1",
		"Cam_Sn":     "pcb_d2000.avi",
		"Channels":   3.0,
		"Height":     1200.0,
		"ImgHandle":  "inmem_e46d6a41,persist_5601a975",
		"ImgName":    "vid-fr-inmemory_persistent,vid-fr-inmemory_persistent",
		"Sample_num": 17,
		"Width":      1920.0,
		"encoding":   "jpg",
		"user_data":  1,
		"influx_ts":  1562051538583313921}

	msgtosend := types.NewMsgEnvelope(msg, nil)
	for {
		err = publisher.Publish(msgtosend)
		if err != nil {
			fmt.Printf("-- Failed to publish message: %v\n", err)
			return
		}
	}

}

func testSubscriber(config map[string]interface{}, topic string) {
	fmt.Println("-- Initializing message bus context")
	client, err := eismsgbus.NewMsgbusClient(config)
	if err != nil {
		fmt.Printf("-- Error initializing message bus context: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Printf("-- Subscribing to topic %s\n", topic)
	subscriber, err := client.NewSubscriber(topic)
	if err != nil {
		fmt.Printf("-- Error subscribing to topic: %v\n", err)
		return
	}
	defer subscriber.Close()

	for {
		select {
		case msg := <-subscriber.MessageChannel:
			fmt.Printf("-- Received Message: %v\n", msg)
		case err := <-subscriber.ErrorChannel:
			fmt.Printf("-- Error receiving message: %v\n", err)
		}
	}

}

func main() {
	pubconfigFile := flag.String("pubconfigFile", "", "JSON configuration file")
	subconfigFile := flag.String("subconfigFile", "", "JSON configuration file")
	topic := flag.String("topic", "", "Subscription topic")

	flag.Parse()

	if *pubconfigFile == "" {
		fmt.Println("-- Publisher Config file must be specified")
		return
	}

	if *subconfigFile == "" {
		fmt.Println("-- Publisher Config file must be specified")
		return
	}

	fmt.Printf("-- Loading Publisher configuration file %s\n", *pubconfigFile)
	pubconfig, err := eismsgbus.ReadJsonConfig(*pubconfigFile)
	if err != nil {
		fmt.Printf("-- Failed to parse config: %v\n", err)
		return
	}

	fmt.Printf("-- Loading Subscriber configuration file %s\n", *subconfigFile)
	subconfig, err := eismsgbus.ReadJsonConfig(*subconfigFile)
	if err != nil {
		fmt.Printf("-- Failed to parse config: %v\n", err)
		return
	}

	done := make(chan bool)
	go testSubscriber(subconfig, *topic)
	go testPublisher(pubconfig, *topic)
	<-done

}
