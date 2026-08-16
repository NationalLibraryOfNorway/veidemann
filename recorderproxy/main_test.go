package main

import "testing"

func TestCacheAddress(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    string
		want    string
		wantErr bool
	}{
		{name: "missing host and port", wantErr: true},
		{name: "missing host", port: "3128", wantErr: true},
		{name: "missing port", host: "cache", wantErr: true},
		{name: "hostname", host: "cache", port: "3128", want: "cache:3128"},
		{name: "IPv6 address", host: "2001:db8::1", port: "3128", want: "[2001:db8::1]:3128"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cacheAddress(tt.host, tt.port)
			if (err != nil) != tt.wantErr {
				t.Fatalf("cacheAddress(%q, %q) error = %v, wantErr %v", tt.host, tt.port, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("cacheAddress(%q, %q) = %q, want %q", tt.host, tt.port, got, tt.want)
			}
		})
	}
}
