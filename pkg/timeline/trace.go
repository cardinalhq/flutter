package timeline

import (
	"fmt"

	"github.com/cardinalhq/flutter/pkg/script"
)

func mergeTrace(rs *script.Script, trace Trace) error {
	for _, variant := range trace.Variants {
		if len(variant.Timeline) == 0 {
			return fmt.Errorf("no segments for trace %s", trace.Name)
		}

		// 	id := makeTraceID(trace, variant)
		// 	generators := generateGeneratorIDs(id, variant.Timeline)
		// 	firstAt := variant.Timeline[0].StartTs.Get()
		// 	lastAt := variant.Timeline[len(variant.Timeline)-1].EndTs.Get()

		// 	if err := addTraceToConfig(rs, id, trace, variant, generators, firstAt, lastAt); err != nil {
		// 		return err
		// 	}

		// 	if err := addTraceTimelineToScript(rs, id, variant.Timeline); err != nil {
		// 		return err
		// 	}
	}
	return nil
}
