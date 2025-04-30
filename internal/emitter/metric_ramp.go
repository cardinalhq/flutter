package emitter

import (
	"errors"
	"time"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/state"
)

type MetricRampSpec struct {
	MetricEmitterSpec `mapstructure:",squash"`
	Start             float64       `mapstructure:"start" yaml:"start" json:"start"`
	Target            float64       `mapstructure:"target" yaml:"target" json:"target"`
	Duration          time.Duration `mapstructure:"duration" yaml:"duration" json:"duration"`
}

type MetricRamp struct {
	spec MetricRampSpec
	at   time.Duration
}

var _ MetricEmitter = (*MetricRamp)(nil)

func NewMetricRamp(at time.Duration, is map[string]any) (*MetricRamp, error) {
	spec := MetricRampSpec{}
	decoder, err := config.NewMapstructureDecoder(&spec)
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(is); err != nil {
		return nil, err
	}
	if spec.Duration <= 0 {
		return nil, errors.New("invalid duration")
	}
	state := MetricRamp{
		spec: spec,
		at:   at,
	}
	return &state, nil
}

func (m *MetricRamp) Reconfigure(at time.Duration, is map[string]any) error {
	oldSpec := m.spec
	oldAt := m.at

	newSpec := oldSpec
	decoder, err := config.NewMapstructureDecoder(&newSpec)
	if err != nil {
		return err
	}
	if err := decoder.Decode(is); err != nil {
		return err
	}
	if newSpec.Duration <= 0 {
		return errors.New("invalid duration")
	}

	if at <= oldAt {
		m.spec = newSpec
		m.at = at
		return nil
	}

	current := intrerpolate(
		oldSpec.Start,
		oldSpec.Target,
		oldAt,
		at,
		oldSpec.Duration,
	)

	m.spec = newSpec
	m.spec.Start = current
	m.at = at

	return nil
}

func (m *MetricRamp) Emit(rs *state.RunState, value float64) float64 {
	v := intrerpolate(m.spec.Start, m.spec.Target, m.at, rs.Now, m.spec.Duration)
	return v + value
}

// intrerpolate linearly interpolates from start → target over the given duration,
// beginning at offset startAt, and evaluated at offset at.
func intrerpolate(start, target float64, startAt, now, duration time.Duration) float64 {
	if duration <= 0 {
		return target
	}
	elapsed := now - startAt
	if elapsed <= 0 {
		return start
	}
	if elapsed >= duration {
		return target
	}
	frac := float64(elapsed) / float64(duration)
	return start + (target-start)*frac
}
