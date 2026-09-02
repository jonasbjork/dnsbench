package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"dnsbench/internal/dnsbench"
)

func main() {
	count := flag.Int("n", dnsbench.DefaultCount, "antal DNS-uppslagningar per server")
	file := flag.String("f", "", "läs DNS-servrar från fil")
	flag.Parse()

	domain := dnsbench.DefaultDomain
	if flag.NArg() > 0 {
		domain = flag.Arg(0)
	}

	if *count < 1 {
		fmt.Fprintln(os.Stderr, "-n måste vara minst 1")
		os.Exit(1)
	}

	servers := dnsbench.DefaultServers
	if *file != "" {
		var err error
		servers, err = dnsbench.ReadServers(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kunde inte läsa %s: %v\n", *file, err)
			os.Exit(1)
		}
	}

	if len(servers) == 0 {
		fmt.Fprintln(os.Stderr, "inga DNS-servrar angivna")
		os.Exit(1)
	}

	fmt.Printf("\nTestar %d DNS-uppslagningar mot %s\n", *count, domain)
	fmt.Printf("Testar %d DNS-servrar parallellt...\n", len(servers))

	start := time.Now()
	results := dnsbench.ProbeServers(servers, domain, *count)
	elapsed := time.Since(start)

	sort.Slice(results, func(i, j int) bool {
		if results[i].Success == 0 {
			return false
		}
		if results[j].Success == 0 {
			return true
		}
		return results[i].Avg < results[j].Avg
	})

	dnsbench.PrintResults(results, *count)
	fmt.Printf("\nTotal testtid: %.2f sekunder\n", elapsed.Seconds())
}
