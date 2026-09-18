package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "http":
		err = runHTTP(os.Args[2:])
	case "coraza":
		err = runCoraza(os.Args[2:])
	case "l4":
		err = runL4(os.Args[2:])
	case "backend":
		err = runBackend(os.Args[2:])
	case "compare":
		err = runCompare(os.Args[2:])
	case "certify":
		err = runCertify(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "wafbench:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `wafbench - repeatable WAF performance harness

Usage:
  wafbench http    [flags]   load-test a running waf-proxy
  wafbench coraza  [flags]   benchmark Coraza/CRS directly, no network/backend
  wafbench l4      [flags]   benchmark TCP connect or TLS-handshake pressure
  wafbench backend [flags]   run a deterministic local benchmark backend
  wafbench compare [flags]   compare JSON results and suggest the next profiling direction
  wafbench certify [flags]   bind real benchmark evidence into a target-aware certification report

Typical flow:
  wafbench backend --listen 127.0.0.1:18081 --response-bytes 16384
  wafbench http --target https://127.0.0.1:8443 --host bench.local --scenario all \
      --duration 15s --warmup 3s --concurrency 64 --pid $(pidof waf-proxy) --out http.json
  wafbench coraza --rules /etc/waf/coraza.conf --scenario all --duration 10s \
      --workers 8 --gomaxprocs 8 --response-inspection inherit --response-bytes 16384 --out coraza.json
  wafbench l4 --target 127.0.0.1:8443 --duration 10s --pid $(pidof waf-proxy) --iface eth0 --out l4.json
  wafbench compare --http http.json --coraza coraza.json --l4 l4.json
  wafbench certify --proxy-baseline proxy.json --coraza-crs coraza.json --target target.json --out performance-certification.json

Run "wafbench <command> -h" for command-specific flags.`)
}
