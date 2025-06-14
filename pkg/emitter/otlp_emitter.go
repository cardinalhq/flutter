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

package emitter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

	"github.com/cardinalhq/flutter/pkg/state"
)

type OTLPEmitter struct {
	client   *http.Client
	endpoint string
	headers  map[string]string
}

func NewOTLPEmitter(client *http.Client, endpoint string, headers map[string]string) (*OTLPEmitter, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return &OTLPEmitter{
		client:   client,
		endpoint: endpoint,
		headers:  headers,
	}, nil
}

func (e *OTLPEmitter) EmitMetrics(ctx context.Context, rs *state.RunState, md pmetric.Metrics) error {
	if md.DataPointCount() == 0 {
		return nil
	}

	req := pmetricotlp.NewExportRequestFromMetrics(md)

	body, err := req.MarshalProto()
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to protobuf: %w", err)
	}

	url := strings.TrimRight(e.endpoint, "/") + "/v1/metrics"
	return e.sendRequest(ctx, url, body)
}

func (e *OTLPEmitter) EmitTraces(ctx context.Context, rs *state.RunState, td ptrace.Traces) error {
	if td.SpanCount() == 0 {
		return nil
	}

	req := ptraceotlp.NewExportRequestFromTraces(td)

	body, err := req.MarshalProto()
	if err != nil {
		return fmt.Errorf("failed to marshal traces to protobuf: %w", err)
	}

	url := strings.TrimRight(e.endpoint, "/") + "/v1/traces"
	return e.sendRequest(ctx, url, body)
}

var ignoreStatusCodes = []int{http.StatusNoContent, http.StatusOK, http.StatusAccepted, http.StatusBadGateway}

func (e *OTLPEmitter) sendRequest(ctx context.Context, url string, body []byte) error {
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

	if !slices.Contains(ignoreStatusCodes, resp.StatusCode) {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("collector returned %s: %s", resp.Status, string(respBody))
	}

	return nil
}
