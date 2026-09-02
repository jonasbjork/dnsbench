package dnsbench_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"dnsbench/internal/dnsbench"
)

func TestBuildDNSQuery(t *testing.T) {
	packet, id, err := dnsbench.BuildDNSQuery("google.se")
	if err != nil {
		t.Fatalf("BuildDNSQuery() error = %v", err)
	}

	if got := binary.BigEndian.Uint16(packet[0:2]); got != id {
		t.Errorf("transaction ID = %d, want %d", got, id)
	}
	if got := binary.BigEndian.Uint16(packet[2:4]); got != 0x0100 {
		t.Errorf("flags = %#x, want %#x", got, 0x0100)
	}
	if got := binary.BigEndian.Uint16(packet[4:6]); got != 1 {
		t.Errorf("question count = %d, want 1", got)
	}

	wantQuestion := []byte{6, 'g', 'o', 'o', 'g', 'l', 'e', 2, 's', 'e', 0, 0, 1, 0, 1}
	if got := packet[12:]; !reflect.DeepEqual(got, wantQuestion) {
		t.Errorf("question = %v, want %v", got, wantQuestion)
	}
}

func TestBuildDNSQueryRejectsLongLabel(t *testing.T) {
	_, _, err := dnsbench.BuildDNSQuery(strings.Repeat("a", 64) + ".se")
	if err == nil {
		t.Fatal("BuildDNSQuery() error = nil, want an error")
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{name: "empty", values: nil, want: 0},
		{name: "odd", values: []float64{9, 1, 5}, want: 5},
		{name: "even", values: []float64{8, 2, 4, 6}, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := append([]float64(nil), tt.values...)
			if got := dnsbench.Median(tt.values); got != tt.want {
				t.Errorf("Median(%v) = %v, want %v", tt.values, got, tt.want)
			}
			if !reflect.DeepEqual(tt.values, original) {
				t.Errorf("Median modified its input: got %v, want %v", tt.values, original)
			}
		})
	}
}

func TestCalculateResult(t *testing.T) {
	result := dnsbench.CalculateResult("1.1.1.1", []float64{10, 20, 30}, 4)

	want := dnsbench.Result{
		Server: "1.1.1.1", Avg: 20, Median: 20, Min: 10, Max: 30,
		Loss: 25, Success: 3, Failed: 1,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("CalculateResult() = %+v, want %+v", result, want)
	}
}

func TestCalculateResultWithNoResponses(t *testing.T) {
	result := dnsbench.CalculateResult("9.9.9.9", nil, 2)
	if result.Success != 0 || result.Failed != 2 || result.Loss != 100 {
		t.Errorf("CalculateResult() = %+v, want 0 successes, 2 failures and 100%% loss", result)
	}
}

func TestReadServers(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "servers.txt")
	content := "# primary\n1.1.1.1\n\n  8.8.8.8  \n# secondary\n9.9.9.9\n"
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := dnsbench.ReadServers(filename)
	if err != nil {
		t.Fatalf("ReadServers() error = %v", err)
	}
	want := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadServers() = %v, want %v", got, want)
	}
}

func TestReadServersReturnsFileError(t *testing.T) {
	_, err := dnsbench.ReadServers(filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil {
		t.Fatal("ReadServers() error = nil, want an error")
	}
}
