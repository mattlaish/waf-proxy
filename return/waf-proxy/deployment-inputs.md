<!-- documentation-review: 2026-09-28; classification: current-generated-return -->
# Deployment inputs

All fixture values are non-production. Deployment owner must provide and verify: management origin and chosen profile (dedicated 443 or same-host 19405), resolved IP allowlist, tenant/source/integration/principal IDs, server certificate chain/CA and SNI, opaque reader credential reference, worker ACL, and optional UI origin. JWT issuer/audience and mTLS client identity are **not applicable** to this WAF opaque-bearer profile. Dashboard owns server-certificate validation/revocation from its worker; WAF owns TLS serving and ingress separation.
