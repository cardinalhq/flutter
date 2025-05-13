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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/script"
)

func TestParseTimeline(t *testing.T) {
	t.Run("valid metric timeline input", func(t *testing.T) {
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
									"target": 100
								},
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"target": 150
								},
								{
									"type": "segment",
									"start_ts": "2400s",
									"end_ts": "2460s",
									"target": 100
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
									"target": 200
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
									"target": 100
								},
								{
									"type": "segment",
									"start_ts": "1800s",
									"end_ts": "2400s",
									"target": 150
								},
								{
									"type": "segment",
									"start_ts": "2400s",
									"end_ts": "2460s",
									"target": 100
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
									"target": 200
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

	t.Run("valid trace timeline input", func(t *testing.T) {
		input := `
		{
  "traces": [
    {
      "ref": "582aa3ba-eaa4-4c19-854c-8e069b52f178",
      "name": "Order Placement Flow",
      "exemplar": {
        "ref": "e7bb97f8-d4f1-45c4-a963-dcd61237d444",
        "name": "POST /checkout",
        "kind": "Client",
        "start_ts": "0ms",
        "duration": "500ms",
        "error": false,
        "resourceAttributes": {
          "service.name": "frontend",
          "k8s.cluster.name": "test-cluster",
          "k8s.namespace.name": "chq-demo-apps"
        },
        "attributes": {
          "url.template": "/checkout",
          "http.request.method": "POST",
          "http.response.status_code": "200"
        },
        "children": [
          {
            "ref": "3fa67f2f-12d9-4072-97d1-498e755111f5",
            "name": "POST /checkout",
            "kind": "Server",
            "start_ts": "5ms",
            "duration": "450ms",
            "error": false,
            "resourceAttributes": {
              "service.name": "checkoutservice",
              "k8s.cluster.name": "test-cluster",
              "k8s.namespace.name": "chq-demo-apps"
            },
            "attributes": {
              "url.template": "/checkout",
              "http.request.method": "POST",
              "http.response.status_code": "200"
            },
            "children": [
              {
                "ref": "7e837c80-57bf-4328-a007-99fcd58aec0d",
                "name": "POST /process_payment",
                "kind": "Client",
                "start_ts": "10ms",
                "duration": "300ms",
                "error": false,
                "resourceAttributes": {
                  "service.name": "checkoutservice",
                  "k8s.cluster.name": "test-cluster",
                  "k8s.namespace.name": "chq-demo-apps"
                },
                "attributes": {
                  "url.template": "/process_payment",
                  "http.request.method": "POST",
                  "http.response.status_code": "200"
                },
                "children": [
                  {
                    "ref": "9f955ecd-27b1-4d1e-adec-b1c80bb86819",
                    "name": "POST /process_payment",
                    "kind": "Server",
                    "start_ts": "15ms",
                    "duration": "250ms",
                    "error": false,
                    "resourceAttributes": {
                      "service.name": "paymentservice",
                      "k8s.cluster.name": "test-cluster",
                      "k8s.namespace.name": "chq-demo-apps"
                    },
                    "attributes": {
                      "url.template": "/process_payment",
                      "http.request.method": "POST",
                      "http.response.status_code": "200"
                    },
                    "children": []
                  }
                ]
              },
              {
                "ref": "aaa1568e-1582-4802-aa85-ab66d60cc4d9",
                "name": "POST /send_confirmation",
                "kind": "Client",
                "start_ts": "12ms",
                "duration": "50ms",
                "error": false,
                "resourceAttributes": {
                  "service.name": "checkoutservice",
                  "k8s.cluster.name": "test-cluster",
                  "k8s.namespace.name": "chq-demo-apps"
                },
                "attributes": {
                  "url.template": "/send_confirmation",
                  "http.request.method": "POST",
                  "http.response.status_code": "200"
                },
                "children": [
                  {
                    "ref": "c7829463-8e7e-4f75-8744-d724ab365f89",
                    "name": "POST /send_confirmation",
                    "kind": "Server",
                    "start_ts": "20ms",
                    "duration": "20ms",
                    "error": false,
                    "resourceAttributes": {
                      "service.name": "emailservice",
                      "k8s.cluster.name": "test-cluster",
                      "k8s.namespace.name": "chq-demo-apps"
                    },
                    "attributes": {
                      "url.template": "/send_confirmation",
                      "http.request.method": "POST",
                      "http.response.status_code": "200"
                    },
                    "children": []
                  }
                ]
              }
            ]
          }
        ]
      },
      "variants": [
        {
          "ref": "37daca41-82ec-4fb7-93b8-724deec6984b",
          "name": "Normal Operations",
          "timeline": [
            {
              "start_ts": "0m",
              "end_ts": "20m",
              "start": 50,
              "target": 50
            }
          ],
          "overrides": {}
        },
        {
          "ref": "ee46ba5c-9fd1-4c14-8650-bac382416f17",
          "name": "Error Operations",
          "timeline": [
            {
              "start_ts": "5m",
              "end_ts": "15m",
              "start": 100,
              "target": 100
            }
          ],
          "overrides": {
            "e7bb97f8-d4f1-45c4-a963-dcd61237d444": {
              "error": true,
              "attributes": {
                "http.response.status_code": "500"
              }
            },
            "3fa67f2f-12d9-4072-97d1-498e755111f5": {
              "error": true,
              "attributes": {
                "http.response.status_code": "500"
              }
            },
            "aaa1568e-1582-4802-aa85-ab66d60cc4d9": {
              "error": true,
              "attributes": {
                "http.response.status_code": "500"
              }
            },
            "c7829463-8e7e-4f75-8744-d724ab365f89": {
              "error": true,
              "attributes": {
                "http.response.status_code": "500"
              }
            }
          }
        },
        {
          "ref": "4a2a5ca9-eb98-4d7c-aaa5-9fa8d77c491a",
          "name": "Slow Operations",
          "timeline": [
            {
              "start_ts": "3m",
              "end_ts": "18m",
              "start": 80,
              "target": 60
            }
          ],
          "overrides": {
            "e7bb97f8-d4f1-45c4-a963-dcd61237d444": {
              "duration": "900ms"
            },
            "3fa67f2f-12d9-4072-97d1-498e755111f5": {
              "duration": "810ms"
            },
            "7e837c80-57bf-4328-a007-99fcd58aec0d": {
              "duration": "740ms"
            },
            "9f955ecd-27b1-4d1e-adec-b1c80bb86819": {
              "duration": "600ms"
            }
          }
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
