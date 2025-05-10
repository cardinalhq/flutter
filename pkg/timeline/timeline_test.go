// Copyright 2025 CardinalHQ, Inc
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

package timeline

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeline(t *testing.T) {
	t.Run("valid timeline input", func(t *testing.T) {
		input := `
		{
			"metrics": [
				{
					"name": "spanmetrics.http_requests_sent",
					"type": "count",
					"resourceAttributes": {
						"service.name": "checkoutservice",
						"k8s.namespace.name": "chq-demo-apps",
						"k8s.cluster.name": "test-cluster"
					},
					"variants": [
						{
							"attributes": {
								"http.request.method": "POST",
								"url.template": "http://paymentservice.chq-demo-apps.svc.cluster.local:9095/process-payment",
								"http.response.status_code": 200,
								"has_error": false
							},
							"timeline": [
								{
									"type": "segment",
									"start_ts": "0s",
									"end_ts": "1800s",
									"median": 100
								},
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"median": 150
								},
								{
									"type": "segment",
									"start_ts": "2400s",
									"end_ts": "2460s",
									"median": 100
								}
							]
						},
						{
							"attributes": {
								"http.request.method": "POST",
								"url.template": "http://paymentservice.chq-demo-apps.svc.cluster.local:9095/process-payment",
								"http.response.status_code": 500,
								"has_error": true
							},
							"timeline": [
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"median": 200
								}
							]
						}
					]
				},
				{
					"name": "spanmetrics.http_requests_received",
					"type": "count",
					"resourceAttributes": {
						"service.name": "paymentservice",
						"k8s.namespace.name": "chq-demo-apps",
						"k8s.cluster.name": "test-cluster"
					},
					"variants": [
						{
							"attributes": {
								"http.request.method": "POST",
								"url.template": "/process-payment",
								"http.response.status_code": 200,
								"has_error": false
							},
							"timeline": [
								{
									"type": "segment",
									"start_ts": "0s",
									"end_ts": "1800s",
									"median": 100
								},
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"median": 150
								},
								{
									"type": "segment",
									"start_ts": "2400s",
									"end_ts": "2460s",
									"median": 100
								}
							]
						},
						{
							"attributes": {
								"http.request.method": "POST",
								"url.template": "/process-payment",
								"http.response.status_code": 500,
								"has_error": true
							},
							"timeline": [
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"median": 200
								}
							]
						}
					]
				}
			]
		}`

		var expected Timeline
		err := json.Unmarshal([]byte(input), &expected)
		assert.NoError(t, err)

		result, err := ParseTimeline([]byte(input))
		assert.NoError(t, err)
		assert.Equal(t, &expected, result)
	})

	t.Run("invalid JSON input", func(t *testing.T) {
		input := `invalid json`

		result, err := ParseTimeline([]byte(input))
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		input := `{}`

		result, err := ParseTimeline([]byte(input))
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Metrics)
	})
}

func TestTimelineConversion(t *testing.T) {
	t.Run("valid timeline conversion", func(t *testing.T) {
		input := `
			{
				"metrics": [
					{
						"name": "spanmetrics.http_requests_sent",
						"type": "sum",
						"variants": [
							{
								"timeline": [
									{
										"start": 50,
										"end_ts": "20m",
										"target": 50,
										"start_ts": "0m"
									}
								],
								"attributes": {
									"has_error": "false",
									"url.template": "/process-payments",
									"span.kind.string": "Client",
									"http.request.method": "POST",
									"http.response.status_code": "200"
								}
							},
							{
								"timeline": [
									{
										"start": 100,
										"end_ts": "15m",
										"target": 100,
										"start_ts": "5m"
									}
								],
								"attributes": {
									"has_error": "true",
									"url.template": "/process-payments",
									"span.kind.string": "Client",
									"http.request.method": "POST",
									"http.response.status_code": "500"
								}
							}
						],
						"resourceAttributes": {
							"service.name": "checkoutservice",
							"k8s.cluster.name": "test-cluster",
							"k8s.namespace.name": "chq-demo-apps"
						}
					}
				]
			}`
		var expected Timeline
		err := json.Unmarshal([]byte(input), &expected)
		require.NoError(t, err)

		result, err := ParseTimeline([]byte(input))
		require.NoError(t, err)

		rscript := script.NewScript()
		err = result.MergeIntoScript(rscript)
		require.NoError(t, err)

		require.NoError(t, rscript.Prepare(&config.Config{}))

		_ = rscript.Dump(os.Stdout)

		assert.Equal(t, 20*time.Minute, rscript.Duration())
	})
}
