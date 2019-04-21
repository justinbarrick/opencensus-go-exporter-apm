// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package jaeger contains an OpenCensus tracing exporter for Jaeger.
package apm // import "contrib.go.opencensus.io/exporter/apm"

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"fmt"
	"go.elastic.co/fastjson"
	apm "go.elastic.co/apm/model"
	"go.opencensus.io/trace"
	"strconv"
)

const defaultServiceName = "OpenCensus"

// NewExporter returns a trace.Exporter implementation that exports
// the collected spans to APM.
func NewExporter(url string) *Exporter {
	return &Exporter{
		Url: url,
		client: &http.Client{},
	}
}

// Exporter is an implementation of trace.Exporter that uploads spans to APM.
type Exporter struct {
	Url string
	client *http.Client
}

var _ trace.Exporter = (*Exporter)(nil)

// ExportSpan exports a SpanData to APM.
func (e *Exporter) ExportSpan(data *trace.SpanData) {}

// Flush waits for exported trace spans to be uploaded.
//
// This is useful if your program is ending and you do not want to lose recent spans.
func (e *Exporter) Flush() {}

// As per the OpenCensus Status code mapping in
//    https://opencensus.io/tracing/span/status/
// the status is OK if the code is 0.
const opencensusStatusCodeOK = 0

func spanDataToAPM(data *trace.SpanData) *apm.Transaction {
	sampled := data.SpanContext.TraceOptions.IsSampled()

	tagsMap := tagsToMap(data.Attributes)

	tags := apm.StringMap{
		apm.StringMapItem{Key: "status.code", Value: fmt.Sprintf("%d", data.Status.Code)},
		apm.StringMapItem{Key: "status.message", Value: data.Status.Message},
	}

	// Ensure that if Status.Code is not OK, that we set the "error" tag on the APM span.
	// See Issue https://github.com/census-instrumentation/opencensus-go/issues/1041
	if data.Status.Code != opencensusStatusCodeOK {
		tags = append(tags, apm.StringMapItem{Key: "error", Value: "true"})
	}

	for key, value := range tagsMap {
		tags = append(tags, apm.StringMapItem{Key: key, Value: value})
	}

	var request *apm.Request

	if tagsMap["http.host"] != "" {
		request = &apm.Request{
			URL: tagsToURL(tagsMap),
			Method: tagsMap["http.method"],
		}

		if tagsMap["http.user_agent"] != "" {
			request.Headers = []apm.Header{
				{
					Key:    "User-Agent",
					Values: []string{tagsMap["http.user_agent"]},
				},
			}
		}
	}

	var response *apm.Response
	if tagsMap["http.status_code"] != "" {
		statusCode, _ := strconv.Atoi(tagsMap["http.status_code"])
		response = &apm.Response{StatusCode: statusCode}
	}

	return &apm.Transaction{
		ID:        apm.SpanID(data.SpanContext.SpanID),
		TraceID:   apm.TraceID(data.SpanContext.TraceID),
		ParentID:  apm.SpanID(data.ParentSpanID),
		Name:      data.Name,
		Timestamp: apm.Time(data.StartTime),
		Duration:  float64(data.EndTime.Sub(data.StartTime)),
		Type:      fmt.Sprintf("%d", data.SpanKind),
		Result:    data.Status.Message,
		SpanCount: apm.SpanCount{
			Dropped: 0,
			Started: data.ChildSpanCount,
		},
		Context: &apm.Context{
			Tags: tags,
			//Service: serviceToAPM(proc),
			Request: request,
			Response: response,
		},
		Sampled: &sampled,
	}
}

func (e *Exporter) sendToAPM(transaction *apm.Transaction) error {
	var transactionEncoded fastjson.Writer
	fastjson.Marshal(&transactionEncoded, transaction)

	var metadata fastjson.Writer
	fastjson.Marshal(&metadata, &apm.Service{
		Name: "apm-gateway",
		Agent: &apm.Agent{
			Name:    "apm-gateway",
			Version: "0.0.1",
		},
	})

	buf := &bytes.Buffer{}
	buf.Write([]byte("{\"metadata\":{\"service\":"))
	buf.Write(metadata.Bytes())
	buf.Write([]byte("}}\n{\"transaction\":"))
	buf.Write(transactionEncoded.Bytes())
	buf.Write([]byte("}\n"))

	log.Println(string(buf.Bytes()))

	resp, err := e.client.Post(e.Url, "application/x-ndjson", buf)

	if err := resp.Body.Close(); err != nil {
		return err
	}

	return err
}

func tagsToMap(attributes map[string]interface{}) map[string]string {
	tags := map[string]string{}
	for k, v := range attributes {
		strValue := ""

		switch value := v.(type) {
		case string:
			strValue = value
		case float64:
			strValue = fmt.Sprintf("%f", value)
		case bool:
			strValue = fmt.Sprintf("%t", value)
		case int64:
			strValue = fmt.Sprintf("%d", value)
		case int32:
			strValue = fmt.Sprintf("%d", value)
		}

		tags[k] = strValue
	}
	return tags
}

func urlToAPM(requestUrl url.URL) apm.URL {
	return apm.URL{
		Full:     requestUrl.String(),
		Protocol: requestUrl.Scheme,
		Hostname: requestUrl.Hostname(),
		Port:     requestUrl.Port(),
		Path:     requestUrl.Path,
		Search:   requestUrl.RawQuery,
		Hash:     requestUrl.Fragment,
	}
}

func tagsToURL(tags map[string]string) apm.URL {
	return urlToAPM(url.URL{
		Scheme: "http",
		Host:   tags["http.host"],
		Path:   tags["http.path"],
	})
}
