package script

import (
	"errors"
	"testing"
	"time"

	"github.com/cardinalhq/flutter/pkg/scriptaction"
)

func TestCalculateDuration(t *testing.T) {
	tests := []struct {
		name          string
		cd            time.Duration
		actions       []scriptaction.ScriptAction
		expected      time.Duration
		expectedError error
	}{
		{
			name: "cd is zero, use last action's At if no To is set",
			cd:   0,
			actions: []scriptaction.ScriptAction{
				{At: 5 * time.Second},
				{At: 10 * time.Second},
			},
			expected:      10 * time.Second,
			expectedError: nil,
		},
		{
			name: "cd is zero, use last action's To if set",
			cd:   0,
			actions: []scriptaction.ScriptAction{
				{At: 5 * time.Second, To: 15 * time.Second},
				{At: 10 * time.Second, To: 20 * time.Second},
			},
			expected:      20 * time.Second,
			expectedError: nil,
		},
		{
			name: "cd is less than last action's At",
			cd:   8 * time.Second,
			actions: []scriptaction.ScriptAction{
				{At: 5 * time.Second},
				{At: 10 * time.Second},
			},
			expected:      0,
			expectedError: errors.New("Duration must be greater than or equal to the last script action time, or set to 0"),
		},
		{
			name: "cd is greater than last action's At",
			cd:   15 * time.Second,
			actions: []scriptaction.ScriptAction{
				{At: 5 * time.Second},
				{At: 10 * time.Second},
			},
			expected:      15 * time.Second,
			expectedError: nil,
		},
		{
			name:          "no actions provided",
			cd:            0,
			actions:       []scriptaction.ScriptAction{},
			expected:      0,
			expectedError: errors.New("no actions provided"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if tt.expectedError == nil || r.(error).Error() != tt.expectedError.Error() {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			result, err := calculateDuration(tt.cd, tt.actions)
			if err != nil && tt.expectedError == nil {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && tt.expectedError != nil {
				t.Errorf("expected error: %v, got nil", tt.expectedError)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
			if result != tt.expected {
				t.Errorf("expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}
