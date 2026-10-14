// private-test is development tooling; it is never shipped in tunnelctl archives.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/privatetest"
)

func main() {
	o := privatetest.Options{}
	flag.StringVar(&o.File, "file", "", "Explicit private input path (never its content)")
	flag.StringVar(&o.Core, "core", app.DefaultCore(), "Approved protected core executable")
	flag.StringVar(&o.ProbeURL, "probe-url", "", "Explicit small HTTPS target without query, credentials or fragment")
	flag.BoolVar(&o.Connect, "connect", false, "Authorize real sequential proxy probes")
	flag.IntVar(&o.MaxSamples, "max-samples", 5, "Maximum input lines, 1–20")
	flag.IntVar(&o.MaxProfiles, "max-profiles", 3, "Maximum tested profiles per source, 1–20")
	flag.IntVar(&o.MaxConnections, "max-connections", 1, "Maximum total sequential proxy requests, 0–5")
	flag.DurationVar(&o.Timeout, "timeout", 30*time.Second, "Total deadline per sample, 1s–60s")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	report, e := privatetest.Run(ctx, o)
	// Do not print flag paths, raw errors, HTTP/core diagnostics or source metadata.
	if json.NewEncoder(os.Stdout).Encode(report) != nil {
		os.Exit(1)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, "Private test harness failed; details withheld. Check options, input protection and cleanup.")
		os.Exit(1)
	}
	if !report.Passed {
		os.Exit(5)
	}
}
