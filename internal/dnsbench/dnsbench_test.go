package dnsbench

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestValidateDNSResponse(t *testing.T) {
	const transactionID = 42

	tests := []struct {
		name    string
		flags   uint16
		id      uint16
		wantErr string
	}{
		{name: "success", flags: 0x8180, id: transactionID},
		{name: "not a response", flags: 0x0100, id: transactionID, wantErr: "response-flagga"},
		{name: "truncated", flags: 0x8380, id: transactionID, wantErr: "avkortat"},
		{name: "NXDOMAIN", flags: 0x8183, id: transactionID, wantErr: "felkod 3"},
		{name: "wrong transaction ID", flags: 0x8180, id: 43, wantErr: "transaction ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := make([]byte, 12)
			binary.BigEndian.PutUint16(response[0:2], tt.id)
			binary.BigEndian.PutUint16(response[2:4], tt.flags)

			err := validateDNSResponse(response, transactionID)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateDNSResponse() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("validateDNSResponse() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDNSResponseRejectsShortPacket(t *testing.T) {
	if err := validateDNSResponse(make([]byte, 11), 42); err == nil {
		t.Fatal("validateDNSResponse() error = nil, want an error")
	}
}
