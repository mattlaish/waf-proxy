# Phase 5 Slice F — Performance Certification

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

This directory documents the evidence contract implemented by `wafbench certify`.
The certification layer **does not generate synthetic sizing numbers** and does not
turn component microbenchmarks into production throughput claims.

## Required full-proxy evidence roles

Run `wafbench http --scenario all --format json` against the same host, workload,
concurrency, GOMAXPROCS and deterministic backend under these configurations:

1. `reverse_proxy_baseline` — TLS reverse proxy with WAF rules disabled.
2. `coraza_crs` — the same TLS reverse proxy with Coraza and the production or
   representative OWASP CRS configuration enabled.
3. `vectorscan_assisted` — optional; the same Coraza/CRS configuration with
   VectorScan acceleration enabled only after the Phase 1 real differential gate
   has passed with zero observed false negatives.

The certification command hashes every supplied result file, checks the recorded
system identity and run shape, and refuses to treat incomparable runs as a
certification result.

## Truth boundary

`wafbench certify` separates three states:

- `evidence_status`: whether the real benchmark inputs are present and comparable;
- `target_status`: whether an explicit approved target was evaluated;
- `status`: production performance certification status.

A report cannot become `PASS` unless real full-proxy evidence is complete **and**
an explicit `waf-proxy-performance-target-v1` target is supplied and satisfied.
If VectorScan evidence is included, a real `wafqualify` report in format
`waf-phase1-vectorscan-differential-v1` with `result=PASS`,
`zero_false_negatives=true`, and `false_negative_count=0` is also mandatory.

Without an approved target the report remains `NOT_RUN`, even when benchmark
files are structurally valid. This prevents measured-but-untargeted data from
being mislabeled as a certified deployment size.

## Example command

```bash
./bin/wafbench certify \
  --proxy-baseline evidence/proxy-baseline.json \
  --coraza-crs evidence/coraza-crs.json \
  --vectorscan evidence/vectorscan-assisted.json \
  --vectorscan-qualification evidence/phase1-qualification.json \
  --target qualification/performance/target.example.json \
  --source-sha256 <SHA256-OF-ARTIFACT-UNDER-TEST> \
  --out performance-certification.json
```

`target.example.json` is deliberately labeled `EXAMPLE_NOT_APPROVED`. Replace its
values with owner-approved SLOs before using it as release evidence.

## Regression comparison

Supply `--previous <prior-performance-certification.json>` to record per-role,
per-scenario RPS and p99 percentage deltas. Regression deltas are evidence only;
they do not create a hidden pass/fail threshold.

## Metrics

The report retains RPS, p50/p95/p99 latency, application throughput in Mbps and
Gbps, network RX/TX Gbps when available, process CPU, CPU microseconds/request,
RSS and host softirq. Load-generator saturation is emitted as a warning.

Application throughput is payload throughput from `wafbench`; it is not Ethernet
line rate. No throughput figure in this repository should be treated as a
production sizing claim unless the corresponding certification report is bound
to the tested artifact and approved target.
