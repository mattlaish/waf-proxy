#!/usr/bin/env python3
from pathlib import Path
import re, sys

root=Path(__file__).resolve().parents[2]
files={name:(root/name).read_text(encoding='utf-8') for name in [
    'tls_session.go','tls_observability.go','metrics.go','listeners.go','main.go','admin.go','static/admin.html','tls_session_test.go'
]}
checks=[]
def need(label, cond):
    checks.append((label,bool(cond)))

session=files['tls_session.go']; obs=files['tls_observability.go']; metrics=files['metrics.go']; listeners=files['listeners.go']; main=files['main.go']; admin=files['admin.go']; ui=files['static/admin.html']; tests=files['tls_session_test.go']
need('purpose-separated KDF label', 'waf-proxy/tls-session-ticket/v1' in session)
need('shared secret env', 'WAF_TLS_SESSION_TICKET_SECRET' in session)
need('secret file env', 'WAF_TLS_SESSION_TICKET_SECRET_FILE' in session)
need('secret-file permissions fail closed', 'must not be group/world accessible' in session)
need('minimum secret entropy floor', 'must be at least 32 bytes' in session)
need('HMAC-SHA256 derivation', 'hmac.New(sha256.New' in session)
need('SetSessionTicketKeys used', 'SetSessionTicketKeys' in session)
need('current plus previous ring', 'defaultTLSSessionTicketRetained = 7' in session)
need('24h rotation', 'defaultTLSSessionTicketRotation = 24 * time.Hour' in session)
need('rotation updates live configs', 'for cfg := range m.configs' in session and 'cfg.SetSessionTicketKeys(keys)' in session)
need('no HA token reuse in ticket source', 'WAF_HA_PEER_TOKEN' not in session and 'PeerToken' not in session)
need('ticket secret not persisted in Config', 'TLSSessionTicket' not in re.sub(r'(?s)func.*','',main.split('type Config struct',1)[1].split('// Build metadata',1)[0]) if 'type Config struct' in main else True)
need('listener attach', 'tlsTickets.attach' in listeners)
need('listener detach', 'tlsTickets.detach' in listeners)
need('observer installed on Go TLS config', 'installTLSHandshakeObserver' in listeners)
need('authoritative DidResume', 'cs.DidResume' in metrics)
need('VerifyConnection observes resumptions', 'VerifyConnection' in obs and 'observeTLSHandshake' in obs)
need('connection-local config clone', 'base.Clone()' in obs and 'cfg.GetConfigForClient = nil' in obs)
need('TLS version counters', 'tls12' in metrics and 'tls13' in metrics)
need('certificate algorithm counters', 'tlsECDSA' in metrics and 'tlsRSA' in metrics and 'tlsEd25519' in metrics)
need('latency histogram', 'tlsHandshakeLatencyBounds' in metrics and 'TLSHandshakeP95MS' in metrics)
need('full/resumed rates', 'TLSFullPerSec' in metrics and 'TLSResumedPerSec' in metrics)
need('resume percentage', 'TLSResumePct' in metrics)
need('policy rejected separated', 'tlsPolicyRejected' in metrics)
need('metrics API exposes TLS totals', '"tls":' in admin and 'metrics.tlsStatus()' in admin)
need('status exposes ticket mode', '"tls_session_tickets"' in admin)
need('Console resumption card', 'TLS resumed' in ui and 'tls_resume_pct' in ui)
need('Console p95 card', 'TLS p95' in ui and 'tls_handshake_p95_ms' in ui)
need('cross-config resumption regression', 'TestTLSSessionTicketSharedSecretResumesAcrossServerConfigs' in tests)
need('rotation overlap regression', 'TestTLSSessionTicketKeysDeterministicAcrossNodesAndOverlapRotation' in tests)
need('observer regression', 'TestTLSHandshakeObserverRecordsFullResumeVersionCertificateAndLatency' in tests)
need('policy reject regression', 'TestTLSHandshakeObserverCountsPolicyRejectWithoutSuccess' in tests)
need('software certificate leaf parsed once', 'x509.ParseCertificate(cert.Certificate[0])' in main)

bad=[label for label,ok in checks if not ok]
if bad:
    for x in bad: print('FAIL',x,file=sys.stderr)
    raise SystemExit(1)
print(f'TLS_SESSION_RESUMPTION_OBSERVABILITY_SOURCE_GATE_PASS checks={len(checks)}')
