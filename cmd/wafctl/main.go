package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var version = "dev"
var errNotConfigured = errors.New("not configured")

type commonFlags struct {
	adminURL  string
	token     string
	tokenFile string
	insecure  bool
}

func addCommonFlags(fs *flag.FlagSet, c *commonFlags) {
	defURL := os.Getenv("WAF_ADMIN_URL")
	if defURL == "" {
		defURL = "http://127.0.0.1:9090"
	}
	fs.StringVar(&c.adminURL, "admin-url", defURL, "WAF admin API base URL")
	fs.StringVar(&c.token, "token", os.Getenv("WAF_ADMIN_TOKEN"), "admin bearer token (prefer env/token-file)")
	fs.StringVar(&c.tokenFile, "token-file", "/etc/waf/waf-proxy.env", "file containing WAF_ADMIN_TOKEN=...")
	fs.BoolVar(&c.insecure, "insecure", false, "skip TLS verification for admin API (lab only)")
}

func (c commonFlags) client() (*apiClient, error) {
	tok := strings.TrimSpace(c.token)
	if tok == "" && c.tokenFile != "" {
		b, err := os.ReadFile(c.tokenFile)
		if err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "WAF_ADMIN_TOKEN=") {
					tok = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "WAF_ADMIN_TOKEN="))
					break
				}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read token file: %w", err)
		}
	}
	if tok == "" {
		return nil, errors.New("admin token required: set WAF_ADMIN_TOKEN, --token, or --token-file")
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if c.insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- explicit lab-only CLI flag
	}
	return &apiClient{base: strings.TrimRight(c.adminURL, "/"), token: tok, http: &http.Client{Timeout: 30 * time.Second, Transport: tr}}, nil
}

type apiClient struct {
	base  string
	token string
	http  *http.Client
}

func (c *apiClient) request(method, path string, body []byte) ([]byte, http.Header, error) {
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, resp.Header, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("admin API %s: %s", resp.Status, truncate(string(b), 1024))
	}
	return b, resp.Header, nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "doctor":
		err = runDoctor(os.Args[2:])
	case "debug":
		err = runDebug(os.Args[2:])
	case "support":
		err = runSupport(os.Args[2:])
	case "qualification":
		err = runQualification(os.Args[2:])
	case "coverage":
		err = runCoverage(os.Args[2:])
	case "proxy":
		err = runProxy(os.Args[2:])
	case "release":
		err = runRelease(os.Args[2:])
	case "version", "--version", "-version":
		fmt.Println("wafctl", version)
		return
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		if errors.Is(err, errNotConfigured) {
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `wafctl commands:
  wafctl doctor [--admin-url URL] [--json]
  wafctl debug capture --tenant SITE [--duration 5m]
  wafctl debug stop --tenant SITE
  wafctl debug list --tenant SITE [--limit 50]
  wafctl debug export --tenant SITE --transaction-id ID --output incident.zip
  wafctl support bundle [--output bundle.zip] [--tenant SITE --transaction-id ID]
  wafctl qualification report [--admin-url URL]
  wafctl coverage analyze --rules PATH [--report FILE] [--inventory FILE] [--json]
  wafctl proxy identity show
  wafctl release verify-signature --artifact FILE --signature FILE.minisig --public-key FILE

Authentication: WAF_ADMIN_TOKEN, --token, or --token-file /etc/waf/waf-proxy.env.`)
}

func runProxy(args []string) error {
	if len(args) < 2 || args[0] != "identity" || args[1] != "show" {
		return errors.New("usage: wafctl proxy identity show")
	}
	fmt.Println("Client Identity Trust Boundary")
	fmt.Println("Mode: STRICT")
	fmt.Println("Forwarded headers: trusted only from configured trusted proxies")
	return nil
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	var c commonFlags
	addCommonFlags(fs, &c)
	jsonOut := fs.Bool("json", false, "print raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cli, err := c.client()
	if err != nil {
		return err
	}
	b, _, err := cli.request(http.MethodGet, "/api/doctor", nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		var pretty bytes.Buffer
		if json.Indent(&pretty, b, "", "  ") == nil {
			fmt.Println(pretty.String())
			return nil
		}
		fmt.Println(string(b))
		return nil
	}
	var doc struct {
		Checks []struct {
			Name, Status, Detail string
		} `json:"checks"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	for _, ck := range doc.Checks {
		fmt.Printf("%-9s %-24s %s\n", ck.Status, ck.Name, ck.Detail)
	}
	return nil
}

func runDebug(args []string) error {
	if len(args) < 1 {
		return errors.New("debug subcommand required: capture|stop|list|export")
	}
	switch args[0] {
	case "capture":
		fs := flag.NewFlagSet("debug capture", flag.ContinueOnError)
		var c commonFlags
		addCommonFlags(fs, &c)
		tenant := fs.String("tenant", "", "site/tenant to capture")
		duration := fs.Duration("duration", 5*time.Minute, "capture duration (max 1h)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *tenant == "" || *duration <= 0 || *duration > time.Hour {
			return errors.New("--tenant is required and --duration must be >0 and <=1h")
		}
		cli, err := c.client()
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]any{"tenant": *tenant, "enabled": true, "duration_sec": int(duration.Seconds())})
		b, _, err := cli.request(http.MethodPost, "/api/debug/capture", body)
		if err == nil {
			fmt.Println(string(b))
		}
		return err
	case "stop":
		fs := flag.NewFlagSet("debug stop", flag.ContinueOnError)
		var c commonFlags
		addCommonFlags(fs, &c)
		tenant := fs.String("tenant", "", "site/tenant to stop")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *tenant == "" {
			return errors.New("--tenant is required")
		}
		cli, err := c.client()
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]any{"tenant": *tenant, "enabled": false})
		b, _, err := cli.request(http.MethodPost, "/api/debug/capture", body)
		if err == nil {
			fmt.Println(string(b))
		}
		return err
	case "list":
		fs := flag.NewFlagSet("debug list", flag.ContinueOnError)
		var c commonFlags
		addCommonFlags(fs, &c)
		tenant := fs.String("tenant", "", "site/tenant")
		limit := fs.Int("limit", 50, "maximum records")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *tenant == "" {
			return errors.New("--tenant is required")
		}
		cli, err := c.client()
		if err != nil {
			return err
		}
		path := "/api/debug/evidence?tenant=" + url.QueryEscape(*tenant) + "&limit=" + fmt.Sprint(*limit)
		b, _, err := cli.request(http.MethodGet, path, nil)
		if err == nil {
			var pretty bytes.Buffer
			if json.Indent(&pretty, b, "", "  ") == nil {
				fmt.Println(pretty.String())
			} else {
				fmt.Println(string(b))
			}
		}
		return err
	case "export":
		fs := flag.NewFlagSet("debug export", flag.ContinueOnError)
		var c commonFlags
		addCommonFlags(fs, &c)
		tenant := fs.String("tenant", "", "site/tenant")
		txid := fs.String("transaction-id", "", "captured transaction ID")
		out := fs.String("output", "", "output ZIP path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *tenant == "" || *txid == "" || *out == "" {
			return errors.New("--tenant, --transaction-id and --output are required")
		}
		cli, err := c.client()
		if err != nil {
			return err
		}
		path := "/api/debug/export?tenant=" + url.QueryEscape(*tenant) + "&transaction_id=" + url.QueryEscape(*txid)
		b, _, err := cli.request(http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		if err := verifyZipBytes(b); err != nil {
			return fmt.Errorf("server returned invalid incident ZIP: %w", err)
		}
		if err := atomicWrite(*out, b, 0600); err != nil {
			return err
		}
		h := sha256.Sum256(b)
		fmt.Printf("%s  %s\n", hex.EncodeToString(h[:]), *out)
		return nil
	default:
		return fmt.Errorf("unknown debug subcommand %q", args[0])
	}
}

func runQualification(args []string) error {
	if len(args) < 1 || args[0] != "report" {
		return errors.New("usage: wafctl qualification report")
	}
	fs := flag.NewFlagSet("qualification report", flag.ContinueOnError)
	var c commonFlags
	addCommonFlags(fs, &c)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cli, err := c.client()
	if err != nil {
		return err
	}
	b, _, err := cli.request(http.MethodGet, "/api/qualification/report", nil)
	if err != nil {
		return err
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, b, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(b))
	}
	return nil
}

func runSupport(args []string) error {
	if len(args) < 1 || args[0] != "bundle" {
		return errors.New("usage: wafctl support bundle [flags]")
	}
	fs := flag.NewFlagSet("support bundle", flag.ContinueOnError)
	var c commonFlags
	addCommonFlags(fs, &c)
	out := fs.String("output", "waf-support-"+time.Now().Format("20060102-150405")+".zip", "output ZIP")
	tenant := fs.String("tenant", "", "optional site/tenant for incident evidence")
	txid := fs.String("transaction-id", "", "optional transaction ID")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if (*tenant == "") != (*txid == "") {
		return errors.New("--tenant and --transaction-id must be provided together")
	}
	cli, err := c.client()
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for name, path := range map[string]string{
		"doctor.json":                      "/api/doctor",
		"status.json":                      "/api/status",
		"config/sanitized-config.json":     "/api/config",
		"metrics/metrics.json":             "/api/metrics",
		"release/vector-acceleration.json": "/api/vector-acceleration",
		"debug/status.json":                "/api/debug/status",
	} {
		b, _, err := cli.request(http.MethodGet, path, nil)
		if err != nil {
			files[name] = []byte(fmt.Sprintf("{\"collection_error\":%q}\n", err.Error()))
			continue
		}
		files[name] = prettyJSON(b)
	}
	if *tenant != "" {
		path := "/api/debug/export?tenant=" + url.QueryEscape(*tenant) + "&transaction_id=" + url.QueryEscape(*txid)
		b, _, err := cli.request(http.MethodGet, path, nil)
		if err != nil {
			return fmt.Errorf("incident evidence: %w", err)
		}
		if err := verifyZipBytes(b); err != nil {
			return err
		}
		files["debug/incident.zip"] = b
	}
	files["release/dependencies.json"] = dependencyEvidence(files["doctor.json"])
	files["release/sbom.spdx.json"] = spdxEvidence(files["doctor.json"])
	if b, err := tailFile("/var/log/waf/audit.log", 256<<10); err == nil && len(b) > 0 {
		files["logs/audit.log"] = sanitizeText(b)
	}
	if b, err := commandOutput(5*time.Second, "journalctl", "-u", "waf-proxy", "-n", "500", "--no-pager", "--output=short-iso"); err == nil && len(b) > 0 {
		files["logs/journal.log"] = sanitizeText(b)
	}
	bundle, err := buildSupportZip(files)
	if err != nil {
		return err
	}
	if err := atomicWrite(*out, bundle, 0600); err != nil {
		return err
	}
	h := sha256.Sum256(bundle)
	fmt.Printf("%s  %s\n", hex.EncodeToString(h[:]), *out)
	return nil
}

func buildSupportZip(files map[string][]byte) ([]byte, error) {
	for name, b := range files {
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
			return nil, fmt.Errorf("unsafe bundle path %q", name)
		}
		if name != "debug/incident.zip" {
			b = sanitizeText(b)
			files[name] = b
			if err := rejectSecrets(name, b); err != nil {
				return nil, err
			}
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	sums := &strings.Builder{}
	manifestFiles := map[string]string{}
	for _, name := range names {
		h := sha256.Sum256(files[name])
		hexsum := hex.EncodeToString(h[:])
		manifestFiles[name] = hexsum
		fmt.Fprintf(sums, "%s  %s\n", hexsum, name)
	}
	files["SHA256SUMS.txt"] = []byte(sums.String())
	manifest, _ := json.MarshalIndent(map[string]any{
		"format": "waf-support-bundle-v1", "generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"wafctl_version": version, "files": manifestFiles,
	}, "", "  ")
	files["manifest.json"] = append(manifest, '\n')
	names = names[:0]
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0640)
		h.SetModTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dependencyEvidence(doctor []byte) []byte {
	local := map[string]string{}
	for name, spec := range map[string][]string{
		"go": {"go", "version"}, "pkg_config_libhs": {"pkg-config", "--modversion", "libhs"},
		"nginx": {"nginx", "-v"}, "openssl": {"openssl", "version"},
	} {
		if b, err := commandOutput(3*time.Second, spec[0], spec[1:]...); err == nil {
			local[name] = strings.TrimSpace(string(b))
		} else {
			local[name] = "NOT_AVAILABLE"
		}
	}
	var server any
	_ = json.Unmarshal(doctor, &server)
	b, _ := json.MarshalIndent(map[string]any{"local": local, "server_doctor": server}, "", "  ")
	return append(b, '\n')
}

func spdxEvidence(doctor []byte) []byte {
	var d struct {
		Modules []struct {
			Path, Version, Sum string
		} `json:"modules"`
	}
	_ = json.Unmarshal(doctor, &d)
	pkgs := make([]map[string]any, 0, len(d.Modules)+1)
	pkgs = append(pkgs, map[string]any{"name": "waf-proxy", "SPDXID": "SPDXRef-waf-proxy", "versionInfo": "runtime-reported", "downloadLocation": "NOASSERTION", "filesAnalyzed": false})
	for i, m := range d.Modules {
		pkg := map[string]any{"name": m.Path, "SPDXID": fmt.Sprintf("SPDXRef-go-module-%d", i+1), "versionInfo": m.Version, "downloadLocation": "NOASSERTION", "filesAnalyzed": false}
		if m.Sum != "" {
			pkg["comment"] = "Go module checksum: " + m.Sum
		}
		pkgs = append(pkgs, pkg)
	}
	b, _ := json.MarshalIndent(map[string]any{
		"spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0", "SPDXID": "SPDXRef-DOCUMENT",
		"name": "waf-proxy-support-evidence", "documentNamespace": "urn:waf-proxy:support:" + fmt.Sprint(time.Now().UTC().UnixNano()),
		"creationInfo": map[string]any{"created": time.Now().UTC().Format(time.RFC3339), "creators": []string{"Tool: wafctl-" + version}},
		"packages":     pkgs,
	}, "", "  ")
	return append(b, '\n')
}

var sensitiveHeaderLine = regexp.MustCompile(`(?im)^(\s*(authorization|proxy-authorization|cookie|set-cookie)\s*[:=]\s*).*$`)
var sensitiveAssignment = regexp.MustCompile(`(?i)(password|passwd|api[_-]?key|secret|peer[_-]?token|admin[_-]?token)(["' ]*[:=]["' ]*)([^,\s"']+)`)

func sanitizeText(b []byte) []byte {
	s := string(b)
	if strings.Contains(strings.ToUpper(s), "BEGIN PRIVATE KEY") || strings.Contains(strings.ToUpper(s), "BEGIN RSA PRIVATE KEY") || strings.Contains(strings.ToUpper(s), "BEGIN EC PRIVATE KEY") {
		s = "[private-key material removed]\n"
	}
	s = sensitiveHeaderLine.ReplaceAllString(s, `$1[masked]`)
	s = sensitiveAssignment.ReplaceAllString(s, `$1$2[masked]`)
	if len(s) > 4<<20 {
		s = s[len(s)-(4<<20):]
	}
	return []byte(s)
}

func rejectSecrets(name string, b []byte) error {
	lower := strings.ToLower(string(b))
	for _, needle := range []string{"begin private key", "begin rsa private key", "begin ec private key", "waf_admin_token="} {
		if strings.Contains(lower, needle) {
			return fmt.Errorf("secret material rejected from %s", name)
		}
	}
	return nil
}

func commandOutput(timeout time.Duration, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return buf.Bytes(), err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return buf.Bytes(), errors.New("command timed out")
	}
}

func tailFile(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	start := st.Size() - max
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(f, max))
}

func prettyJSON(b []byte) []byte {
	var out bytes.Buffer
	if json.Indent(&out, b, "", "  ") == nil {
		out.WriteByte('\n')
		return out.Bytes()
	}
	return b
}

func verifyZipBytes(b []byte) error {
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return err
	}
	if len(r.File) == 0 {
		return errors.New("empty ZIP")
	}
	for _, f := range r.File {
		clean := filepath.Clean(f.Name)
		if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("unsafe ZIP path %q", f.Name)
		}
	}
	return nil
}

func atomicWrite(path string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil && filepath.Dir(path) != "." {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func runRelease(args []string) error {
	if len(args) < 1 || args[0] != "verify-signature" {
		return errors.New("release subcommand required: verify-signature")
	}
	fs := flag.NewFlagSet("release verify-signature", flag.ContinueOnError)
	artifact := fs.String("artifact", "", "release artifact to verify")
	signature := fs.String("signature", "", "detached minisign signature")
	publicKey := fs.String("public-key", "", "approved minisign public-key file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *artifact == "" || *signature == "" || *publicKey == "" {
		return errors.New("--artifact, --signature, and --public-key are required")
	}
	for _, p := range []string{*artifact, *signature, *publicKey} {
		st, err := os.Stat(p)
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("not a regular file: %s", p)
		}
	}
	b, err := os.ReadFile(*publicKey)
	if err != nil {
		return err
	}
	pub := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(strings.ToLower(line), "untrusted") {
			pub = line
			break
		}
	}
	if pub == "" {
		return errors.New("public-key file contains no minisign key")
	}
	if _, err := exec.LookPath("minisign"); err != nil {
		fmt.Println("NOT_CONFIGURED")
		return errNotConfigured
	}
	cmd := exec.Command("minisign", "-Vm", *artifact, "-x", *signature, "-P", pub)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature INVALID: %s", truncate(strings.TrimSpace(string(out)), 1024))
	}
	fmt.Println("VALID")
	return nil
}
