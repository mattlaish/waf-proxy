package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func writeJSON(path string, v any) error {
	b, _ := json.MarshalIndent(v, "", "  ")
	return os.WriteFile(path, b, 0640)
}

func bundle(out string, kind string) error {
	tmp, err := os.MkdirTemp("", "waf-bundle-*")
	if err != nil { return err }
	defer os.RemoveAll(tmp)
	if err := writeJSON(filepath.Join(tmp,"manifest.json"), map[string]any{
		"type": kind, "created": time.Now().UTC(), "go": runtime.Version(),
	}); err != nil { return err }
	z, err := os.Create(out); if err != nil { return err }
	defer z.Close()
	w:=zip.NewWriter(z); defer w.Close()
	return filepath.Walk(tmp, func(p string, i os.FileInfo, err error) error {
		if err != nil || i.IsDir() { return err }
		f, e:=w.Create(filepath.Base(p)); if e!=nil{return e}
		b,e:=os.ReadFile(p); if e!=nil{return e}
		_,e=f.Write(b); return e
	})
}

func main() {
	if len(os.Args)<2 { fmt.Println("wafctl <doctor|debug export|support bundle>"); return }
	switch os.Args[1] {
	case "doctor":
		_ = writeJSON("waf-doctor.json", map[string]any{
			"runtime":runtime.Version(),
			"status":"NOT_RUN",
			"coraza":"NOT_RUN",
			"vectorscan":"NOT_RUN",
		})
		fmt.Println("waf doctor: NOT_RUN dependency qualification")
	case "debug":
		if len(os.Args)>2 && os.Args[2]=="export" {
			fs:=flag.NewFlagSet("export",flag.ExitOnError)
			out:=fs.String("output","incident.zip","output bundle")
			fs.Parse(os.Args[3:])
			if err:=bundle(*out,"debug-incident");err!=nil{panic(err)}
		}
	case "support":
		if len(os.Args)>2 && os.Args[2]=="bundle" {
			if err:=bundle("waf-support.zip","support");err!=nil{panic(err)}
		}
	}
}
