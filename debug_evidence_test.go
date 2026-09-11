package main

import (
 "testing"
 "time"
)

func TestDebugEvidenceCaptureMaskTTLFoundation(t *testing.T){
 d:=NewDebugEvidenceCapture(2)
 d.Enable(true)
 d.SetTTL(time.Minute)
 d.Capture("tx1", map[string]any{"authorization":"secret","rule":"942100"})
 if err:=d.Export(t.TempDir()+"/bundle.zip"); err!=nil { t.Fatal(err) }
}
