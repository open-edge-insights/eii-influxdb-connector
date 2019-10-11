/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dbManager

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	common "IEdgeInsights/InfluxDBConnector/common"

	"github.com/golang/glog"
)

// InfluxSubCtx structure
type InfluxSubCtx struct {
	SbInfo       common.SubScriptionInfo
	pData        chan string
	OutInterface common.OutPutInterface
}

const (
	maxPointsBuffered = 100
	influxCaPath      = "/etc/ssl/ca/ca_certificate.pem"
	influxCertPath    = "/etc/ssl/influxdb/influxdb_server_certificate.pem"
	influxKeyPath     = "/etc/ssl/influxdb/influxdb_server_key.pem"
)

func (subCtx *InfluxSubCtx) handlePointData(workerID int) {
	glog.Infof("Go routine %v for subscription started", workerID)
	for {
		// Wait for data in point data buffer
		buf := <-subCtx.pData

		if common.Profiling == true {
			temp := strings.Fields(buf)
			fields := temp[1] + ",ts_idbconn_pub_queue_exit=" + strconv.FormatInt((time.Now().UnixNano()/1e6), 10)
			buf = temp[0] + " " + fields + " " + temp[2]
			//glog.Infof("modified reqBody is %v", buf)
		}

		subCtx.OutInterface.Write([]byte(buf))
	}
}

func (subCtx *InfluxSubCtx) httpHandlerFunc(w http.ResponseWriter, req *http.Request) {

	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		glog.Errorf("Error in reading the data: %v", err)
	}

	var ts_temp1, ts_temp2 int64

	if common.Profiling == true {
		temp := strings.Fields(string(reqBody))
		ts_temp1 = time.Now().UnixNano() / 1e6
		fields := temp[1] + ",ts_idbconn_pub_entry=" + strconv.FormatInt(ts_temp1, 10)
		reqBody = []byte(temp[0] + " " + fields + " " + temp[2])
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8") // normal header
	w.Header().Set("Strict-Transport-Security", "max-age=1024000; includeSubDomains")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Received a POST request\n"))

	if common.Profiling == true {
		//glog.Infof("reqBody is %v", string(reqBody))
		temp := strings.Fields(string(reqBody))
		ts_temp2 = time.Now().UnixNano() / 1e6
		diff := ts_temp2 - ts_temp1
		fields := temp[1] + ",ts_idbconn_pub_queue_entry=" + strconv.FormatInt(ts_temp2, 10)
		fields += ",ts_idbconn_influx_respose_write=" + strconv.FormatInt(diff, 10)
		reqBody = []byte(temp[0] + " " + fields + " " + temp[2])
		//glog.Infof("modified reqBody is %v", string(reqBody))
	}

	select {
	case subCtx.pData <- string(reqBody):
	default:
		glog.Infof("Discarding the point. Stream generation faster than Publish!")
	}
}

func (subCtx *InfluxSubCtx) startServer(devMode bool) {
	var dstAddr string
	var err error
	if subCtx.SbInfo.Host != "" {
		dstAddr = subCtx.SbInfo.Host + ":" + subCtx.SbInfo.Port
	} else {
		dstAddr = ":" + subCtx.SbInfo.Port
	}

	// Make the channel for handling point data
	subCtx.pData = make(chan string, maxPointsBuffered)
	for workerID := 0; workerID < subCtx.SbInfo.Worker; workerID++ {
		go subCtx.handlePointData(workerID)
	}

	// Start the HTTP server handler
	http.HandleFunc("/", subCtx.httpHandlerFunc)
	if devMode {
		err = http.ListenAndServe(dstAddr, nil)
	} else {

		serverCert, err := ioutil.ReadFile(influxCertPath)
		if err != nil {
			glog.Errorf("%v", err)
			os.Exit(-1)
		}
		serverKey, err := ioutil.ReadFile(influxKeyPath)
		if err != nil {
			glog.Errorf("%v", err)
			os.Exit(-1)
		}
		servTLSCert, err := tls.X509KeyPair(serverCert, serverKey)
		if err != nil {
			glog.Errorf("invalid key pair: %v", err)
			os.Exit(-1)
		}

		// Create a CA certificate pool and add cert.pem to it
		caCert, err := ioutil.ReadFile(influxCaPath)
		if err != nil {
			glog.Errorf("%v", err)
			os.Exit(-1)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Create the TLS Config with the CA pool and enable Client certificate validation
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{servTLSCert},
			ClientCAs:          caCertPool,
			ClientAuth:         tls.VerifyClientCertIfGiven,
			InsecureSkipVerify: false,
		}

		tlsConfig.BuildNameToCertificate()

		// Create  a Server instance to listen on port 61971 with the TLS config
		server := &http.Server{
			Addr:      			dstAddr,
			ReadTimeout:    	60 * time.Second,
			ReadHeaderTimeout:  60 * time.Second,
			WriteTimeout:    	60 * time.Second,
			IdleTimeout:    	60 * time.Second,
			TLSConfig: 			tlsConfig,
			MaxHeaderBytes: 	1 << 20,
		}
		err = server.ListenAndServeTLS(influxCertPath, influxKeyPath)

	}

	if err != nil {
		glog.Errorf("Error in connection to client due to: %v\n", err)
		os.Exit(-1)
	}
}
