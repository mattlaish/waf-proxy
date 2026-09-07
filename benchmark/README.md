# wafbench performance harness

`cmd/wafbench` is the repeatable benchmark/profiling tool for this repository. It is intentionally split into a network/full-proxy test and a direct Coraza/CRS test so low production traffic does not prevent useful performance decisions.

## Build

```bash
go build -o ./bin/wafbench ./cmd/wafbench
```

The build uses the same Coraza module version as waf-proxy. For a meaningful `coraza` run, the host must have the configured CRS files (the shipped `coraza.conf` references `/etc/waf/crs/...`).

## 1. Deterministic backend

Run this on the WAF/backend test host:

```bash
./bin/wafbench backend --listen 127.0.0.1:18081 --response-bytes 16384
```

Point a dedicated benchmark pool/site at this backend. Do not benchmark a production backend if the goal is to isolate WAF cost.

## 2. Full HTTP/WAF benchmark

For CPU-per-request comparison, `wafbench http` must run on the **WAF host** because `--pid` and `--iface` read local `/proc`. Pin the generator to CPUs outside the WAF CPU set (for example with `taskset`) so generator CPU does not become the bottleneck. If you move the load generator to another host, the RPS/latency numbers remain useful, but this command can no longer sample the remote WAF PID; do not feed such a run into the CPU-share comparison unless equivalent WAF-host CPU metrics are collected separately.

```bash
./bin/wafbench http \
  --target https://127.0.0.1:8443 \
  --host bench.local \
  --scenario all \
  --duration 15s \
  --warmup 3s \
  --concurrency 64 \
  --gomaxprocs 8 \
  --pid "$(pidof waf-proxy)" \
  --iface eth0 \
  --insecure \
  --format json \
  --out http-baseline.json
```

Scenarios: `clean-get`, `json-1k`, `json-16k`, `json-64k`, `json-256k`, `sqli-64k`, `xss-64k`, `traversal`, or `all`.

The tool reports application-payload Mbps, not Ethernet line rate. `--pid` is required for waf-proxy CPU/RSS and CPU microseconds/request. Linux `--iface` adds RX/TX Mbps and PPS. Host busy/softirq comes from `/proc/stat`. Generator CPU is recorded separately; if it approaches the selected generator `GOMAXPROCS`, move it to more/different CPUs before trusting the WAF ceiling.

## 3. Coraza-only benchmark

This removes NIC, TCP, TLS, Go reverse proxy and backend cost. DetectionOnly is the default so hostile corpus requests continue through rule evaluation instead of stopping at the first disruptive action. Audit logging is disabled by default to isolate inspection CPU. The 256 KiB request case may cross `SecRequestBodyInMemoryLimit` and exercise temporary-file body handling depending on the active config; treat that scenario as production body-processing cost, not pure regex cost.

```bash
./bin/wafbench coraza \
  --rules /etc/waf/coraza.conf \
  --scenario all \
  --duration 10s \
  --warmup 2s \
  --workers 8 \
  --gomaxprocs 8 \
  --response-inspection inherit \
  --response-bytes 16384 \
  --format json \
  --out coraza-baseline.json
```

To isolate phase-4 response cost:

```bash
./bin/wafbench coraza --rules /etc/waf/coraza.conf --scenario json-64k --response-inspection off --out response-off.json
./bin/wafbench coraza --rules /etc/waf/coraza.conf --scenario json-64k --response-inspection on  --out response-on.json
```

For profiles:

```bash
./bin/wafbench coraza --rules /etc/waf/coraza.conf --scenario sqli-64k \
  --cpuprofile cpu.pprof --heapprofile heap.pprof

go tool pprof -http=:8088 ./bin/wafbench cpu.pprof
```

## 4. L3/L4 pressure baseline

Use legal TCP connect/close traffic to see whether accept/softirq/PPS becomes a significant cost before adding XDP. This does not spoof source IPs or generate raw SYN floods. Against a TLS listener, prefer `--tls` so the server does not spend the test logging intentional handshake EOF errors; for a pure connect/close baseline, use a dedicated plaintext benchmark listener.

```bash
./bin/wafbench l4 \
  --target 127.0.0.1:8443 \
  --duration 10s \
  --concurrency 128 \
  --pid "$(pidof waf-proxy)" \
  --iface eth0 \
  --format json \
  --out l4-baseline.json
```

Add `--tls --server-name bench.local --insecure` to measure handshake pressure separately; plain TCP is the cleaner XDP/L4 baseline.

## 5. XDP vs regex acceleration comparison

```bash
./bin/wafbench compare --http http-baseline.json --coraza coraza-baseline.json --l4 l4-baseline.json
```

The tool uses a deliberately simple heuristic: the Coraza-share median is calculated from **clean GET + clean JSON scenarios only**; malicious scenarios are displayed as worst-case information but do not drive the baseline median. If direct Coraza CPU/request is at least ~35% of full waf-proxy CPU/request, regex/Coraza acceleration is the stronger next experiment. If Coraza share is lower while host softirq is >=5% and aggregate sampled packet rate is >=100k PPS, XDP is the stronger experiment. Otherwise it recommends profiling TLS/proxy/backend before a dataplane rewrite.

This heuristic is a decision aid, not a benchmark claim. Use CPU profiles and representative traffic before committing to VectorScan/Hyperscan or XDP.

## Scaling matrix

Run the Coraza benchmark at 1/4/8 cores rather than only at maximum CPU:

```bash
for n in 1 4 8; do
  ./bin/wafbench coraza --rules /etc/waf/coraza.conf --scenario all \
    --workers "$n" --gomaxprocs "$n" --duration 10s --format json --out "coraza-${n}c.json"
done
```

For the full proxy, combine `GOMAXPROCS`/CPU affinity on the waf-proxy process with the same HTTP corpus. Keep load generator and backend CPU outside the WAF CPU set where possible.


## Safety / test-environment boundary

The hostile corpus is intended for a dedicated benchmark WAF/site and deterministic backend. Do not point this suite at a production application unless you explicitly intend to generate SQLi/XSS/traversal requests against it. The L4 mode uses ordinary TCP connections only; it does not spoof addresses or emit raw SYN floods.
