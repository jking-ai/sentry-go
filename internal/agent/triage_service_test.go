package agent

import (
	"testing"
)

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain json",
			input: `{"root_cause":"bad deploy"}`,
			want:  `{"root_cause":"bad deploy"}`,
		},
		{
			name:  "json in markdown fence",
			input: "```json\n{\"root_cause\":\"bad deploy\"}\n```",
			want:  `{"root_cause":"bad deploy"}`,
		},
		{
			name:  "json in generic fence",
			input: "```\n{\"root_cause\":\"bad deploy\"}\n```",
			want:  `{"root_cause":"bad deploy"}`,
		},
		{
			name:  "json with leading whitespace",
			input: "   \n  {\"root_cause\":\"bad deploy\"}\n  ",
			want:  `{"root_cause":"bad deploy"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanJSONResponse(tt.input)
			if got != tt.want {
				t.Errorf("cleanJSONResponse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("truncate = %q, want %q", got, "hello...")
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("truncate = %q, want %q", got, "hi")
	}
}