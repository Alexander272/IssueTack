package main

import (
	"net/http"
	"testing"
	"time"
)

func TestIsUnboundContext(t *testing.T) {
	p := &Plugin{}
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "bound false", body: `{"data":{"bound":false}}`, want: true},
		{name: "bound true", body: `{"data":{"bound":true,"realmId":"x"}}`, want: false},
		{name: "missing bound", body: `{"data":{}}`, want: false},
		{name: "empty body", body: ``, want: false},
		{name: "error body", body: `{"code":500}`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.isUnboundContext([]byte(tt.body)); got != tt.want {
				t.Errorf("isUnboundContext(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestContextCacheLifecycle(t *testing.T) {
	p := &Plugin{}

	if _, ok := p.cachedContext("ch1"); ok {
		t.Fatal("cache should be empty initially")
	}

	header := http.Header{"Content-Type": []string{"application/json"}}
	body := []byte(`{"data":{"bound":false}}`)
	p.storeContext("ch1", http.StatusOK, header, body)

	ent, ok := p.cachedContext("ch1")
	if !ok {
		t.Fatal("expected cached entry for ch1")
	}
	if ent.status != http.StatusOK {
		t.Errorf("status = %d, want %d", ent.status, http.StatusOK)
	}

	// Истёкший кэш не возвращается.
	ent.ts = time.Now().Add(-contextNegCacheTTL - time.Second)
	p.contextCache["ch1"] = ent
	if _, ok := p.cachedContext("ch1"); ok {
		t.Fatal("expired entry should not be returned")
	}

	// Пустой channelID не кэшируется.
	p.storeContext("", http.StatusOK, header, body)
	if len(p.contextCache) != 1 {
		t.Errorf("expected 1 entry, got %d", len(p.contextCache))
	}
}