/*
Copyright (c) 2019 Intel Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

Explicit permissions are required to publish, distribute, sublicense, and/or sell copies of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package dbManager

import (
        //"crypto/tls"
	//"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"

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
	influxCaPath    = "/etc/ssl/ca/ca_certificate.pem"
        influxCertPath = "/etc/ssl/influxdb/influxdb_server_certificate.pem"
	influxKeyPath     = "/etc/ssl/influxdb/influxdb_server_key.pem"
)

func (subCtx *InfluxSubCtx) handlePointData() {

	for {
		// Wait for data in point data buffer
		buf := <-subCtx.pData
		subCtx.OutInterface.Write([]byte(buf))
	}
}

func (subCtx *InfluxSubCtx) httpHandlerFunc(w http.ResponseWriter, req *http.Request) {

	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		glog.Errorf("Error in reading the data: %v", err)
	}

	w.Write([]byte("Received a POST request\n"))

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
	go subCtx.handlePointData()

	// Start the HTTP server handler
	http.HandleFunc("/", subCtx.httpHandlerFunc)
	if devMode {
		err = http.ListenAndServe(dstAddr, nil)
	} else {
                //TODO Enable Mutual TLS
                /*
                serverCert, err := ioutil.ReadFile(influxCertPath)
                if err != nil {
                    glog.Errorf("%v", err)
                }
                serverKey, err := ioutil.ReadFile(influxKeyPath)
                if err != nil {
                    glog.Errorf("%v", err)
                }
                servTLSCert, err := tls.X509KeyPair(serverCert, serverKey)
                if err != nil {
	             glog.Errorf("invalid key pair: %v", err)
                }
                // Create a CA certificate pool and add cert.pem to it
                caCert, err := ioutil.ReadFile(influxCaPath)
                if err != nil {
                    glog.Errorf("%v", err)
                }
                caCertPool := x509.NewCertPool()
                caCertPool.AppendCertsFromPEM(caCert)

                // Create the TLS Config with the CA pool and enable Client certificate validation
                tlsConfig := &tls.Config{
                    Certificates: []tls.Certificate{servTLSCert},
                    ClientCAs: caCertPool,
                    ClientAuth: tls.RequireAndVerifyClientCert,
                }

                tlsConfig.BuildNameToCertificate()

                // Create  a Server instance to listen on port 61971 with the TLS config
                server := &http.Server{
                    Addr:      dstAddr,
                    TLSConfig: tlsConfig,
                }
		err = server.ListenAndServeTLS(influxCertPath, influxKeyPath)
                */
                err = http.ListenAndServeTLS(dstAddr, influxCertPath, influxKeyPath, nil)
	}

	if err != nil {
		glog.Errorf("Error in connection to client due to: %v\n", err)
		os.Exit(-1)
	}
}
