package mattermost

import (
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
)

func TestSourceGuardAllow(t *testing.T) {
	tests := []struct {
		name      string
		configure []string
		remote    string
		want      bool
	}{
		{
			name:      "single IP match",
			configure: []string{"10.0.0.1"},
			remote:    "10.0.0.1:58321",
			want:      true,
		},
		{
			name:      "single IP mismatch",
			configure: []string{"10.0.0.1"},
			remote:    "10.0.0.2:58321",
			want:      false,
		},
		{
			name:      "CIDR inside",
			configure: []string{"192.168.5.0/24"},
			remote:    "192.168.5.130:9000",
			want:      true,
		},
		{
			name:      "CIDR outside",
			configure: []string{"192.168.5.0/24"},
			remote:    "10.0.0.5:9000",
			want:      false,
		},
		{
			name:      "range lower bound included",
			configure: []string{"192.168.5.200-192.168.5.255"},
			remote:    "192.168.5.200:4587",
			want:      true,
		},
		{
			name:      "range upper bound included",
			configure: []string{"192.168.5.200-192.168.5.255"},
			remote:    "192.168.5.255:4587",
			want:      true,
		},
		{
			name:      "range inside",
			configure: []string{"192.168.5.200-192.168.5.255"},
			remote:    "192.168.5.231:4587",
			want:      true,
		},
		{
			name:      "range below lower bound",
			configure: []string{"192.168.5.200-192.168.5.255"},
			remote:    "192.168.5.199:4587",
			want:      false,
		},
		{
			name:      "remote without port",
			configure: []string{"10.0.0.1"},
			remote:    "10.0.0.1",
			want:      true,
		},
		{
			name:      "invalid remote",
			configure: []string{"10.0.0.1"},
			remote:    "not-an-ip",
			want:      false,
		},
		{
			name:      "multiple entries",
			configure: []string{"10.0.0.1", "192.168.1.0/24", "172.16.0.5-172.16.0.9"},
			remote:    "172.16.0.7:1234",
			want:      true,
		},
		{
			name:      "malformed entries are skipped",
			configure: []string{"garbage", "999.999.1.1", "10.0.0.1"},
			remote:    "10.0.0.1:1",
			want:      true,
		},
		{
			name:      "empty list rejects everything",
			configure: nil,
			remote:    "10.0.0.1:1",
			want:      false,
		},
		{
			name:      "whitespace entries rejected",
			configure: []string{"   ", ""},
			remote:    "10.0.0.1:1",
			want:      false,
		},
		{
			name:      "ipv6 single",
			configure: []string{"2001:db8::1"},
			remote:    "[2001:db8::1]:4711",
			want:      true,
		},
		{
			name:      "ipv6 range",
			configure: []string{"2001:db8::1-2001:db8::10"},
			remote:    "[2001:db8::5]:4711",
			want:      true,
		},
		{
			name:      "ipv6 range outside",
			configure: []string{"2001:db8::1-2001:db8::10"},
			remote:    "[2001:db8::11]:4711",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newSourceGuard(config.MattermostConfig{AllowedServerIPs: tt.configure})
			if got := g.allow(tt.remote); got != tt.want {
				t.Errorf("allow(%q) = %v, want %v", tt.remote, got, tt.want)
			}
		})
	}
}
