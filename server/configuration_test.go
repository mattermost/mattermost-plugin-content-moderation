package main

import (
	"reflect"
	"testing"
)

func TestConfiguration_ExcludedUserSet(t *testing.T) {
	tests := []struct {
		name          string
		excludedUsers string
		expected      map[string]struct{}
	}{
		{
			name:          "empty string",
			excludedUsers: "",
			expected:      map[string]struct{}{},
		},
		{
			name:          "whitespace only",
			excludedUsers: "   ",
			expected:      map[string]struct{}{},
		},
		{
			name:          "single user",
			excludedUsers: "user1",
			expected:      map[string]struct{}{"user1": {}},
		},
		{
			name:          "multiple users",
			excludedUsers: "user1,user2,user3",
			expected:      map[string]struct{}{"user1": {}, "user2": {}, "user3": {}},
		},
		{
			name:          "users with spaces",
			excludedUsers: " user1 , user2 , user3 ",
			expected:      map[string]struct{}{"user1": {}, "user2": {}, "user3": {}},
		},
		{
			name:          "users with empty entries",
			excludedUsers: "user1,,user2,   ,user3",
			expected:      map[string]struct{}{"user1": {}, "user2": {}, "user3": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &configuration{
				ExcludedUsers: tt.excludedUsers,
			}
			result := c.ExcludedUserSet()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExcludedUserSet() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfiguration_ThresholdValue(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
		expected  int
		wantError bool
	}{
		{
			name:      "valid threshold",
			threshold: "5",
			expected:  5,
			wantError: false,
		},
		{
			name:      "zero threshold",
			threshold: "0",
			expected:  0,
			wantError: false,
		},
		{
			name:      "empty threshold",
			threshold: "",
			expected:  0,
			wantError: true,
		},
		{
			name:      "invalid threshold - not a number",
			threshold: "abc",
			expected:  0,
			wantError: true,
		},
		{
			name:      "invalid threshold - float",
			threshold: "5.5",
			expected:  0,
			wantError: true,
		},
		{
			name:      "negative threshold",
			threshold: "-1",
			expected:  -1,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &configuration{
				Threshold: tt.threshold,
			}
			result, err := c.ThresholdValue()
			if tt.wantError {
				if err == nil {
					t.Errorf("ThresholdValue() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ThresholdValue() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ThresholdValue() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}
