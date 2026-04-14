package peers

import (
	"testing"
)

func TestParsePeer(t *testing.T) {
	tests := []struct {
		input    string
		wantOk   bool
		wantName string
		wantAddr string
		wantPort int
	}{
		{
			input:    "ntavelis@192.168.1.64:46789",
			wantOk:   true,
			wantName: "ntavelis",
			wantAddr: "192.168.1.64",
			wantPort: 46789,
		},
		{
			input:    "192.168.1.64:46789",
			wantOk:   true,
			wantName: "192.168.1.64:46789",
			wantAddr: "192.168.1.64",
			wantPort: 46789,
		},
		{
			input:    "192.168.1.100:46789",
			wantOk:   true,
			wantName: "192.168.1.100:46789",
			wantAddr: "192.168.1.100",
			wantPort: 46789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			peer, ok := parsePeer(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("parsePeer(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if peer.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", peer.Name, tt.wantName)
			}
			if peer.Addr != tt.wantAddr {
				t.Errorf("Addr = %q, want %q", peer.Addr, tt.wantAddr)
			}
			if peer.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", peer.Port, tt.wantPort)
			}
		})
	}
}
