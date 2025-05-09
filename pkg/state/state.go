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

package state

import (
	"math/rand/v2"
	"time"
)

type RunState struct {
	Now           time.Duration
	Wallclock     time.Time
	Duration      time.Duration
	RND           *rand.Rand
	CurrentAction int
}

func NewRunState(duration time.Duration, seed uint64) *RunState {
	return &RunState{
		Duration: duration,
		RND:      MakeRNG(seed),
	}
}

func MakeRNG(seed uint64) *rand.Rand {
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	return rand.New(rand.NewPCG(seed, seed))
}
