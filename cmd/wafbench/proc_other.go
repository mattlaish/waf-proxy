//go:build !linux

package main

type procSample struct{}

func sampleProc(pid int, iface string) (procSample, error)            { return procSample{}, nil }
func enrichProc(r *Result, before, after procSample, elapsed float64) {}

func platformDetails() (string, string) { return "", "" }
