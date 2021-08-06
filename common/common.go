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

package common

// DbCredential structure
type DbCredential struct {
	Username  string
	Password  string
	Database  string
	Retention string
	Host      string
	Port      string
	Ssl       string
	Verifyssl string
}

// SubScriptionInfo structure
type SubScriptionInfo struct {
	DbName string
	Host   string
	Port   string
	Worker int
}

// Filter Interface
type Filter interface {
	GetAttribute(data []byte) (string, error)
}

// OutPutInterface interface
type OutPutInterface interface {
	Write(data []byte)
}

// InsertInterface interface
type InsertInterface interface {
	Write(data []byte, topic string)
}

// PubEndPoint structure
type PubEndPoint struct {
	Name string
}

// Clients structure
type Clients struct {
	Name string `json:"name"`
}

// ReqEndPoint structure
type ReqEndPoint struct {
	Name     string
	EndPoint string
}

// AppConfig structure
type AppConfig struct {
	DevMode   bool
	PubWorker int64
	SubWorker int64
}

// SubEndPoint structure
type SubEndPoint struct {
	Measurement string
}

// Profiling variable
var Profiling bool = false
