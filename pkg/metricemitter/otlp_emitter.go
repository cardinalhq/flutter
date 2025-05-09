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

package metricemitter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cardinalhq/flutter/pkg/state"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

type OTLPMetricEmitter struct {
	client   *http.Client
	endpoint string
	headers  map[string]string
}

func NewOTLPMetricEmitter(client *http.Client, endpoint string, headers map[string]string) (*OTLPMetricEmitter, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return &OTLPMetricEmitter{
		client:   client,
		endpoint: endpoint,
		headers:  headers,
	}, nil
}

func (e *OTLPMetricEmitter) Emit(ctx context.Context, rs *state.RunState, md pmetric.Metrics) error {
	if md.DataPointCount() == 0 {
		return nil
	}

	req := pmetricotlp.NewExportRequestFromMetrics(md)

	body, err := req.MarshalProto()
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to protobuf: %w", err)
	}

	url := strings.TrimRight(e.endpoint, "/") + "/v1/metrics"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	for k, v := range e.headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("collector returned %s: %s", resp.Status, string(respBody))
	}

	return nil
}
