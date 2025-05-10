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

package brokenwing

import (
	"errors"
	"fmt"
)

// Custom error types
var (
	ErrInvalidMetricName = errors.New("invalid metric name")
	ErrNoGenerators      = errors.New("no generators specified for metric gauge")
	ErrUnknownGenerator  = errors.New("unknown generator")
)

type DecodeError struct {
	Name string
	Err  error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("unable to decode MetricGaugeSpec for %q: %v", e.Name, e.Err)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}
