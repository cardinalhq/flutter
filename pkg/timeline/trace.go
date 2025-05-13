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
	"fmt"
	"time"

	"github.com/cardinalhq/flutter/pkg/script"
)

func mergeTrace(rs *script.Script, trace Trace) error {
	for _, variant := range trace.Variants {
		if len(variant.Timeline) == 0 {
			return fmt.Errorf("no segments for trace %s", trace.Name)
		}

		id := makeTraceID(trace, variant)
		firstAt := variant.Timeline[0].StartTs.Get()
		lastAt := variant.Timeline[len(variant.Timeline)-1].EndTs.Get()

		if err := addTraceToConfig(rs, id, trace, variant, firstAt, lastAt); err != nil {
			return err
		}

		// if err := addTraceTimelineToScript(rs, id, variant.Timeline); err != nil {
		// 	return err
		// }
	}
	return nil
}

func makeTraceID(trace Trace, variant TraceVariant) string {
	return fmt.Sprintf("%s-%s", trace.Name, variant.Name)
}

func addTraceToConfig(rs *script.Script, id string, trace Trace, variant TraceVariant, firstAt, endAt time.Duration) error {
	// action := scriptaction.ScriptAction{
	// 	At:   firstAt,
	// 	To:   endAt,
	// 	Name: id,
	// 	Type: "trace",
	// 	Spec: specToMap(traceproducer.Trace{
	// 		Name: id,
	// 	}),
	// }

	return nil
}
