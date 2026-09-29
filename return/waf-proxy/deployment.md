# Deployment status: DO NOT ENABLE

There is no executable deployment procedure for this slice. The reader is
unwired and no real `binding.json` exists. Do not publish a new management
listener, mint a production token, or alter WAF service startup from this
prototype alone.

Before deployment: confirm profile A dedicated hostname/TCP443 or profile B
same hostname/TCP19405, concrete management bind IP, TLS server name and CA,
`source_instance_id`, `tenant_id`, worker CIDR/ACL, credential custody,
independent reader-failure behavior, and explicit approval of exported WAF
asset/detection/policy/health metadata. Run root build and A01–A14, then
stage default-off on a test VM. Revoke a deployed machine principal by setting
its `revoked` field or removing it in the protected reader config; rotate with
at most 24-hour overlap. Those steps are design intent, not tested procedure.

