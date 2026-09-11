# Phase 2 VectorScan Coverage Expansion Report

Date: 2026-09-11 (Asia/Taipei)

## Baseline

`waf-proxy-phase0-phase1-qualification-attempt-2026-09-11.zip`

## Owner direction

Phase 2 source implementation was explicitly requested even though Phase 0 and Phase 1 production qualification are still blocked on the packaging host. This report therefore distinguishes implementation evidence from production qualification evidence.

## Implemented coverage

- ordered exact transform pipelines after mandatory `t:none`;
- `lowercase`, `uppercase`, `trim`, `trimLeft`, `trimRight`, `removeNulls`, `replaceNulls`, `compressWhitespace`, `removeWhitespace`, `length`;
- `QUERY_STRING`;
- `SERVER_NAME`;
- fixed-name `REQUEST_HEADERS:name`, including Coraza-connector parity for `Host` and `Transfer-Encoding`.

Still Coraza-only: ARGS/body variables, chains, negated operators, multi-variable selectors, dynamic macros, URL/path/HTML/JS/CSS decoding or normalization, and any non-allow-listed transform.

## Safety behavior

Coraza remains authoritative. The semantic adapter version is bumped so persisted learning fingerprints are invalidated and affected groups re-enter Learning. Existing zero-false-negative and FAILSAFE behavior is unchanged.

## Executed tests

- isolated `internal/vectoraccel` unit suite: PASS;
- focused Phase 2 transform/source tests: PASS;
- isolated `go vet`: PASS;
- isolated `go test -race`: PASS;
- full repository test: BLOCKED by Go 1.25 module requirement on local Go 1.23.2.

## NOT_RUN production gates

- Phase 2 `realcoraza` parity gate;
- real libvectorscan scan;
- representative production CRS differential replay;
- production zero-observed-false-negative qualification.

Phase 2 is therefore `IMPLEMENTED_TESTING_DEFERRED / QUALIFICATION_REQUIRED`, not production-qualified.

## Coverage increment 2

Additional implemented sources:

- `REQUEST_URI_RAW` — mirrors the exact URI string passed by the Coraza Go HTTP connector to `ProcessURI`;
- `REQUEST_LINE` — mirrors Coraza's `method + " " + uri + " " + protocol` construction;
- `REQUEST_BASENAME` — mirrors the parsed request path and last `/` or `\\` separator behavior;
- `REMOTE_ADDR` / `REMOTE_PORT` — mirror the Coraza connector's final-colon split of `req.RemoteAddr`, after the WAF's existing trusted-proxy resolver has normalized it for all downstream consumers.

Additional exact transforms:

- `base64Encode` — standard Base64 encoding of input bytes;
- `hexEncode` — lowercase hexadecimal encoding of input bytes.

The semantic adapter version was bumped again, forcing newly changed groups to re-enter Learning. `base64Decode`, `hexDecode`, `md5`, `sha1`, URL/path/HTML/JS/CSS normalization/decoding, ARGS/body, chains, negation, multi-variable selectors and dynamic macros remain Coraza-only.

Executed validation: isolated `internal/vectoraccel` unit/focused/vet/race PASS. Full repository is BLOCKED on this host by the Go 1.25 requirement. Expanded real-Coraza parity, real libvectorscan and production CRS differential/zero-false-negative gates remain NOT_RUN.
