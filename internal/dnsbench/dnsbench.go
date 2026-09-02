// Package dnsbench innehåller logiken för att mäta svarstider mot DNS-servrar.
package dnsbench

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var DefaultServers = []string{
	"1.1.1.1",
	"8.8.8.8",
	"9.9.9.9",
}

const (
	DefaultDomain = "google.se"
	DefaultCount  = 10
	Timeout       = 2 * time.Second
)

type Result struct {
	Server  string
	Avg     float64
	Median  float64
	Min     float64
	Max     float64
	Loss    float64
	Success int
	Failed  int
}

func BuildDNSQuery(domain string) ([]byte, uint16, error) {
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" {
		return nil, 0, fmt.Errorf("ogiltig domän: tomt namn")
	}

	id := uint16(rand.Intn(65536))
	packet := make([]byte, 12)

	binary.BigEndian.PutUint16(packet[0:2], id)
	binary.BigEndian.PutUint16(packet[2:4], 0x0100)
	binary.BigEndian.PutUint16(packet[4:6], 1)

	for _, part := range strings.Split(domain, ".") {
		if part == "" {
			return nil, 0, fmt.Errorf("ogiltig domän: tom label")
		}
		if len(part) > 63 {
			return nil, 0, fmt.Errorf("ogiltig domän: label för lång")
		}
		if len(packet)-12+len(part)+2 > 255 {
			return nil, 0, fmt.Errorf("ogiltig domän: namnet för långt")
		}

		packet = append(packet, byte(len(part)))
		packet = append(packet, []byte(part)...)
	}

	packet = append(packet, 0)
	packet = binary.BigEndian.AppendUint16(packet, 1)
	packet = binary.BigEndian.AppendUint16(packet, 1)

	return packet, id, nil
}

func QueryDNS(server, domain string) (float64, error) {
	packet, transactionID, err := BuildDNSQuery(domain)
	if err != nil {
		return 0, err
	}

	conn, err := net.DialTimeout("udp", net.JoinHostPort(server, "53"), Timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(Timeout)); err != nil {
		return 0, err
	}

	start := time.Now()
	if _, err := conn.Write(packet); err != nil {
		return 0, err
	}

	response := make([]byte, 4096)
	n, err := conn.Read(response)
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(start)

	if err := validateDNSResponse(response[:n], transactionID); err != nil {
		return 0, err
	}

	return float64(elapsed.Microseconds()) / 1000.0, nil
}

func validateDNSResponse(response []byte, transactionID uint16) error {
	if len(response) < 12 {
		return fmt.Errorf("ogiltigt DNS-svar")
	}
	if binary.BigEndian.Uint16(response[0:2]) != transactionID {
		return fmt.Errorf("fel transaction ID")
	}

	flags := binary.BigEndian.Uint16(response[2:4])
	if flags&0x8000 == 0 {
		return fmt.Errorf("ogiltigt DNS-svar: response-flagga saknas")
	}
	if flags&0x0200 != 0 {
		return fmt.Errorf("avkortat DNS-svar")
	}
	if rcode := flags & 0x000f; rcode != 0 {
		return fmt.Errorf("DNS-servern returnerade felkod %d", rcode)
	}

	return nil
}

func Median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	middle := len(copyValues) / 2

	if len(copyValues)%2 == 0 {
		return (copyValues[middle-1] + copyValues[middle]) / 2
	}
	return copyValues[middle]
}

func ProbeServer(server, domain string, count int) Result {
	if count <= 0 {
		return Result{Server: server}
	}

	times := make([]float64, 0, count)

	for i := 0; i < count; i++ {
		queryTime, err := QueryDNS(server, domain)
		if err == nil {
			times = append(times, queryTime)
		}
	}

	return CalculateResult(server, times, count)
}

func CalculateResult(server string, times []float64, count int) Result {
	if count <= 0 {
		return Result{Server: server}
	}

	success := len(times)
	failed := count - success
	result := Result{
		Server:  server,
		Success: success,
		Failed:  failed,
		Loss:    float64(failed) / float64(count) * 100,
	}

	if success == 0 {
		return result
	}

	result.Min = times[0]
	result.Max = times[0]
	var sum float64

	for _, value := range times {
		sum += value
		if value < result.Min {
			result.Min = value
		}
		if value > result.Max {
			result.Max = value
		}
	}

	result.Avg = sum / float64(success)
	result.Median = Median(times)
	return result
}

func ProbeServers(servers []string, domain string, count int) []Result {
	results := make(chan Result, len(servers))
	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			results <- ProbeServer(server, domain, count)
		}(server)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make([]Result, 0, len(servers))
	for result := range results {
		collected = append(collected, result)
	}
	return collected
}

func ReadServers(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		servers = append(servers, line)
	}
	return servers, nil
}

func PrintResults(results []Result, count int) {
	fmt.Println()
	fmt.Printf("%-18s %10s %10s %10s %10s %9s %9s\n", "DNS server", "Avg", "Median", "Min", "Max", "Loss", "Svar")
	fmt.Println(strings.Repeat("-", 83))

	for _, result := range results {
		if result.Success == 0 {
			fmt.Printf("%-18s %10s %10s %10s %10s %8.1f%% %4d/%d\n", result.Server, "-", "-", "-", "-", result.Loss, result.Success, count)
			continue
		}

		fmt.Printf("%-18s %7.2f ms %7.2f ms %7.2f ms %7.2f ms %8.1f%% %4d/%d\n", result.Server, result.Avg, result.Median, result.Min, result.Max, result.Loss, result.Success, count)
	}
}
