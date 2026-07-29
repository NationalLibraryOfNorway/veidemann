package main

import "testing"

func TestRewrite(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL and extra",
			input: "https://example.com/ job-1",
			want:  "OK store-id=v1|job-1|https://example.com/\n",
		},
		{
			name:  "multiple extras are preserved",
			input: "https://example.com/ job-1 crawl-1",
			want:  "OK store-id=v1|job-1 crawl-1|https://example.com/\n",
		},
		{
			name:  "missing extra",
			input: "https://example.com/",
			want:  "ERR\n",
		},
		{
			name:  "unset extra",
			input: "https://example.com/ -",
			want:  "ERR\n",
		},
		{
			name:  "empty request",
			input: "",
			want:  "BH message=empty-request\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewrite(tt.input); got != tt.want {
				t.Fatalf("rewrite(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
