#!/usr/bin/env python3
"""Provision OWI-1.0 opaque reader tokens without storing plaintext tokens.

The raw token is printed once. The registry stores only a domain-separated
SHA-256 digest plus principal/tenant/scope/expiry metadata.
"""
from __future__ import annotations
import argparse, datetime as dt, hashlib, json, os, secrets, sys
from pathlib import Path

PURPOSE = b"waf-proxy/dashboard-reader-token/v1\x00"
DEFAULT_SCOPES = [
    "dashboard:read:health", "dashboard:read:capabilities", "dashboard:read:assets",
    "dashboard:read:detections", "dashboard:read:policies", "dashboard:read:health-observations",
]
ALL_SCOPES = set(DEFAULT_SCOPES)

def now(): return dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
def z(t): return t.isoformat().replace("+00:00", "Z")
def parse(s): return dt.datetime.fromisoformat(s.replace("Z", "+00:00"))
def digest(token): return hashlib.sha256(PURPOSE + token.encode()).hexdigest()

def load(path):
    if not path.exists(): return {"version":1,"tokens":[]}
    if path.is_symlink(): raise SystemExit("registry must not be a symlink")
    if path.stat().st_mode & 0o077: raise SystemExit("registry must be owner-only")
    obj=json.loads(path.read_text())
    if obj.get("version")!=1 or not isinstance(obj.get("tokens"),list): raise SystemExit("invalid registry")
    return obj

def save(path,obj):
    path.parent.mkdir(parents=True,exist_ok=True)
    tmp=path.with_name("."+path.name+".tmp-"+secrets.token_hex(4))
    fd=os.open(tmp,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    try:
        with os.fdopen(fd,"w") as f: json.dump(obj,f,indent=2); f.write("\n"); f.flush(); os.fsync(f.fileno())
        os.replace(tmp,path)
        dfd=os.open(path.parent,os.O_RDONLY); os.fsync(dfd); os.close(dfd)
    finally:
        try: os.unlink(tmp)
        except FileNotFoundError: pass

def validate_scopes(vals):
    bad=[v for v in vals if v not in ALL_SCOPES]
    if bad: raise SystemExit("unsupported scopes: "+", ".join(bad))
    return sorted(set(vals))

def create_entry(args, token, issued):
    expires=issued+dt.timedelta(days=args.days)
    return {"id":args.id,"principal":args.principal,"tenant_id":args.tenant,
            "digest_sha256":digest(token),"scopes":validate_scopes(args.scope or DEFAULT_SCOPES),
            "issued_at":z(issued),"expires_at":z(expires),"revoked_at":None}

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--registry",type=Path,required=True)
    sp=ap.add_subparsers(dest="cmd",required=True)
    for name in ("create","rotate"):
        p=sp.add_parser(name);p.add_argument("--id",required=True);p.add_argument("--principal",required=True);p.add_argument("--tenant",required=True);p.add_argument("--scope",action="append");p.add_argument("--days",type=int,default=90,choices=range(1,91))
    p=sp.add_parser("revoke");p.add_argument("--id",required=True)
    sp.add_parser("list")
    args=ap.parse_args(); obj=load(args.registry); ts=now()
    if args.cmd in ("create","rotate"):
        if any(x["id"]==args.id for x in obj["tokens"]): raise SystemExit("token id already exists")
        if args.cmd=="rotate":
            # Rotation overlap is bounded to 24h for older active credentials of
            # the same principal/tenant. The new raw token is still shown once.
            overlap=ts+dt.timedelta(hours=24)
            for x in obj["tokens"]:
                if x.get("principal")==args.principal and x.get("tenant_id")==args.tenant and x.get("revoked_at") is None:
                    try:
                        if parse(x["expires_at"])>overlap: x["expires_at"]=z(overlap)
                    except Exception: pass
        token=secrets.token_hex(32)
        obj["tokens"].append(create_entry(args,token,ts));save(args.registry,obj)
        print(token)
    elif args.cmd=="revoke":
        found=False
        for x in obj["tokens"]:
            if x.get("id")==args.id: x["revoked_at"]=z(ts);found=True
        if not found: raise SystemExit("token id not found")
        save(args.registry,obj)
    else:
        safe=[{k:v for k,v in x.items() if k!="digest_sha256"} for x in obj["tokens"]]
        print(json.dumps({"version":1,"tokens":safe},indent=2))
if __name__=="__main__": main()
