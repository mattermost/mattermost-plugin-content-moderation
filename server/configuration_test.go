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
		name            string
		moderatorType   string
		azureThreshold  string
		agentsThreshold string
		expected        int
		wantError       bool
	}{
		{
			name:           "valid azure threshold",
			moderatorType:  "azure",
			azureThreshold: "5",
			expected:       5,
			wantError:      false,
		},
		{
			name:            "valid agents threshold",
			moderatorType:   "agents",
			agentsThreshold: "4",
			expected:        4,
			wantError:       false,
		},
		{
			name:           "zero azure threshold",
			moderatorType:  "azure",
			azureThreshold: "0",
			expected:       0,
			wantError:      false,
		},
		{
			name:           "empty azure threshold",
			moderatorType:  "azure",
			azureThreshold: "",
			expected:       0,
			wantError:      true,
		},
		{
			name:            "empty agents threshold",
			moderatorType:   "agents",
			agentsThreshold: "",
			expected:        0,
			wantError:       true,
		},
		{
			name:           "invalid azure threshold - not a number",
			moderatorType:  "azure",
			azureThreshold: "abc",
			expected:       0,
			wantError:      true,
		},
		{
			name:           "invalid azure threshold - float",
			moderatorType:  "azure",
			azureThreshold: "5.5",
			expected:       0,
			wantError:      true,
		},
		{
			name:           "negative azure threshold",
			moderatorType:  "azure",
			azureThreshold: "-1",
			expected:       -1,
			wantError:      false,
		},
		{
			name:          "unknown moderator type",
			moderatorType: "unknown",
			expected:      0,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &configuration{
				Type:            tt.moderatorType,
				AzureThreshold:  tt.azureThreshold,
				AgentsThreshold: tt.agentsThreshold,
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
