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

package apm

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	apm "go.elastic.co/apm/model"
	"go.opencensus.io/trace"
	"sort"
	"net/url"
)

func Test_spanDataToAPM(t *testing.T) {
	now := time.Now()

	keyValue := "value"
	doubleValue := float64(123.456)
	statusMessage := "error"
	boolTrue := true

	tests := []struct {
		name string
		data *trace.SpanData
		want *apm.Transaction
	}{
		{
			name: "no parent",
			data: &trace.SpanData{
				SpanContext: trace.SpanContext{
					TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
					TraceOptions: trace.TraceOptions(1),
				},
				Name:      "/foo",
				StartTime: now,
				EndTime:   now,
				Attributes: map[string]interface{}{
					"double": doubleValue,
					"key":    keyValue,
				},
				Status: trace.Status{Code: trace.StatusCodeUnknown, Message: "error"},
			},
			want: &apm.Transaction{
				Name: "/foo",
				ID:   apm.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				TraceID: apm.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				Result: statusMessage,
				Timestamp: apm.Time(now),
				Sampled: &boolTrue,
				Context: &apm.Context{
					Tags: apm.StringMap{
						apm.StringMapItem{"status.code", "2"},
						apm.StringMapItem{"status.message", "error"},
						apm.StringMapItem{"error", "true"},
						apm.StringMapItem{"key", "value"},
						apm.StringMapItem{"double", "123.456000"},
					},
				},
				Type: "0",
			},
		},
		{
			name: "parent",
			data: &trace.SpanData{
				SpanContext: trace.SpanContext{
					TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
					TraceOptions: trace.TraceOptions(1),
				},
				ParentSpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				Name:      "/foo",
				StartTime: now,
				EndTime:   now,
				Attributes: map[string]interface{}{
					"double": doubleValue,
					"key":    keyValue,
				},
				Status: trace.Status{Code: trace.StatusCodeUnknown, Message: "error"},
			},
			want: &apm.Transaction{
				Name: "/foo",
				ID:   apm.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				TraceID: apm.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				ParentID:  apm.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				Result: statusMessage,
				Timestamp: apm.Time(now),
				Sampled: &boolTrue,
				Context: &apm.Context{
					Tags: apm.StringMap{
						apm.StringMapItem{"status.code", "2"},
						apm.StringMapItem{"status.message", "error"},
						apm.StringMapItem{"error", "true"},
						apm.StringMapItem{"key", "value"},
						apm.StringMapItem{"double", "123.456000"},
					},
				},
				Type: "0",
			},
		},
		{
			name: "http request",
			data: &trace.SpanData{
				SpanContext: trace.SpanContext{
					TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
					TraceOptions: trace.TraceOptions(1),
				},
				Name:      "/foo",
				StartTime: now,
				EndTime:   now,
				Attributes: map[string]interface{}{
					"double": doubleValue,
					"key":    keyValue,
					"http.host": "google.com:8080",
					"http.status_code": "200",
					"http.path": "/",
					"http.method": "GET",
					"http.user_agent": "curl/1.4",
				},
				Status: trace.Status{Code: trace.StatusCodeUnknown, Message: "error"},
			},
			want: &apm.Transaction{
				Name: "/foo",
				ID:   apm.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
				TraceID: apm.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				Result: statusMessage,
				Timestamp: apm.Time(now),
				Sampled: &boolTrue,
				Context: &apm.Context{
					Tags: apm.StringMap{
						apm.StringMapItem{"status.code", "2"},
						apm.StringMapItem{"status.message", "error"},
						apm.StringMapItem{"error", "true"},
						apm.StringMapItem{"key", "value"},
						apm.StringMapItem{"double", "123.456000"},
						apm.StringMapItem{"http.host", "google.com:8080"},
						apm.StringMapItem{"http.status_code", "200"},
						apm.StringMapItem{"http.path", "/"},
						apm.StringMapItem{"http.method", "GET"},
						apm.StringMapItem{"http.user_agent", "curl/1.4"},
					},
					Request: &apm.Request{
						URL: apm.URL{
							Full: "http://google.com:8080/",
							Protocol: "http",
							Hostname: "google.com",
							Port: "8080",
							Path: "/",
						},
						Headers: []apm.Header{
							{"User-Agent", []string{"curl/1.4"}},
						},
						Method: "GET",
					},
					Response: &apm.Response{
						StatusCode: 200,
					},
				},
				Type: "0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spanDataToAPM(tt.data)

			sort.Slice(got.Context.Tags, func(i, j int) bool {
				return got.Context.Tags[i].Key < got.Context.Tags[j].Key
			})
			sort.Slice(tt.want.Context.Tags, func(i, j int) bool {
				return tt.want.Context.Tags[i].Key < tt.want.Context.Tags[j].Key
			})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTagsToURL(t *testing.T) {
	parsed, _ := url.Parse("http://google.com:8080/hello")
	assert.Equal(t, tagsToURL(map[string]string{
		"http.host": "google.com:8080",
		"http.path": "/hello",
	}), urlToAPM(*parsed))
}
