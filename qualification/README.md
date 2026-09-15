# Phase 1 VectorScan Production Qualification

`run-phase1-qualification.sh` is the real differential gate. It must run on a
Linux release/target host with Go >=1.25, Coraza v3.7.0 dependencies, real
`libvectorscan`/`libhs`, and the production or representative CRS rules file.

The runner executes the same native VectorScan group scanner used by the WAF,
then executes a real Coraza transaction in DetectionOnly and reads final
`MatchedRules()` after `ProcessLogging()`. Only Coraza matches in the current
conservative VectorScan-eligible rule set are compared; unsupported rules stay
Coraza-only by design.

PASS requires both:

1. observed false negatives = 0; and
2. at least the configured minimum number of eligible Coraza match events.

A native scan error or any false negative is FAIL. Missing Go/libhs/rules, zero
eligible rules, or insufficient eligible-match evidence is BLOCKED/NOT_RUN, not
PASS. Replace or extend `corpus/default.jsonl` with representative production
traffic before using this as production qualification evidence.
