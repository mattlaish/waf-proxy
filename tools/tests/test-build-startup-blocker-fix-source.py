#!/usr/bin/env python3
from pathlib import Path
import os
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
read = lambda p: (ROOT / p).read_text(encoding="utf-8")
checks = []

def req(cond, msg):
    if not cond:
        raise SystemExit("BUILD_STARTUP_BLOCKER_FIX_SOURCE_GATE_FAIL: " + msg)
    checks.append(msg)

admin = read("admin.go")
ui = read("static/admin.html")
reg = read("admin_route_registration_test.go")
build = read("build.sh")
ci = read(".github/workflows/ci.yml")

# Build blocker: errors.New users must have the standard-library import.
req(re.search(r'(?m)^\s*"errors"\s*$', admin) is not None, 'admin.go imports standard-library errors')
req(admin.count('errors.New(') >= 2, 'admin.go errors.New validation paths retained')

# Startup blocker: exception toggle must not overlap operation_id/mode.
new_route = 'POST /api/security/schema/enforcement/exceptions/{exception_id}/toggle'
old_route = 'POST /api/security/schema/enforcement/exceptions/{exception_id}'
req(new_route in admin, 'explicit positive-schema exception toggle route registered')
req(old_route + '"' not in admin, 'ambiguous legacy exception toggle route absent')
req('/api/security/schema/enforcement/exceptions/${encodeURIComponent(b.dataset.id)}/toggle' in ui,
    'shipping Console uses explicit exception toggle route')

# Startup regression: handler construction itself must be tested for panic.
req('TestAdminHandlerRouteRegistrationNoPanic' in reg, 'admin route registration panic regression test exists')
req('(&adminServer{}).handler()' in reg, 'route regression constructs full Admin handler')
req('recover()' in reg, 'route regression fails on ServeMux registration panic')

# Route inventory must be unique at the literal METHOD+path level.
route_text = admin + "\n" + read("update.go")
routes = re.findall(r'mux\.HandleFunc\("([A-Z]+) ([^"\\]+)"', route_text)
req(len(routes) >= 120, 'Admin/update route inventory remains complete')
req(len(routes) == len(set(routes)), 'Admin/update METHOD+path registrations remain unique')

# Literal uniqueness is not enough for Go 1.22+ ServeMux: two distinct wildcard
# patterns can still conflict. Register the exact source inventory with the
# standard library in a standalone temporary program so module dependencies
# and the repository go.mod cannot mask a startup-fatal ambiguity.
patterns = [f"{method} {path}" for method, path in routes]
with tempfile.TemporaryDirectory(prefix="waf-route-gate-") as td:
    gofile = Path(td) / "main.go"
    lines = [
        'package main',
        'import "net/http"',
        'func main() {',
        'm := http.NewServeMux()',
    ]
    for pattern in patterns:
        lines.append(f'm.HandleFunc(`{pattern}`, func(http.ResponseWriter, *http.Request) {{}})')
    lines.append('}')
    gofile.write_text("\n".join(lines) + "\n", encoding="utf-8")
    env = os.environ.copy()
    env["GOTOOLCHAIN"] = "local"
    proc = subprocess.run(["go", "run", str(gofile)], cwd=td, env=env,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    req(proc.returncode == 0, 'Go stdlib ServeMux accepts all Admin/update route patterns without ambiguity')

# Gate is part of both local and hosted build paths.
req('test-build-startup-blocker-fix-source.py' in build, 'build.sh runs blocker-fix source gate')
req('test-build-startup-blocker-fix-source.py' in ci, 'CI runs blocker-fix source gate')

print(f"BUILD_STARTUP_BLOCKER_FIX_SOURCE_GATE_PASS checks={len(checks)} routes={len(routes)}")
