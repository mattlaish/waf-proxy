//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type procSample struct {
	ProcTicks    uint64
	TotalTicks   uint64
	BusyTicks    uint64
	SoftIRQTicks uint64
	RSSKiB       uint64
	NetRXBytes   uint64
	NetTXBytes   uint64
	NetRXPackets uint64
	NetTXPackets uint64
}

func sampleProc(pid int, iface string) (procSample, error) {
	var s procSample
	if pid > 0 {
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			return s, err
		}
		text := string(b)
		end := strings.LastIndex(text, ")")
		if end < 0 {
			return s, fmt.Errorf("malformed /proc/%d/stat", pid)
		}
		fields := strings.Fields(text[end+1:])
		if len(fields) < 13 {
			return s, fmt.Errorf("short /proc/%d/stat", pid)
		}
		ut, _ := strconv.ParseUint(fields[11], 10, 64)
		st, _ := strconv.ParseUint(fields[12], 10, 64)
		s.ProcTicks = ut + st
		status, _ := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				ff := strings.Fields(line)
				if len(ff) >= 2 {
					s.RSSKiB, _ = strconv.ParseUint(ff[1], 10, 64)
				}
				break
			}
		}
	}
	stat, err := os.Open("/proc/stat")
	if err != nil {
		return s, err
	}
	defer stat.Close()
	scanner := bufio.NewScanner(stat)
	if scanner.Scan() {
		f := strings.Fields(scanner.Text())
		if len(f) >= 8 && f[0] == "cpu" {
			vals := make([]uint64, 0, 8)
			for i, x := range f[1:] {
				if i >= 8 {
					break
				} // guest/guest_nice are already accounted in user/nice
				v, _ := strconv.ParseUint(x, 10, 64)
				vals = append(vals, v)
				s.TotalTicks += v
			}
			if len(vals) >= 7 {
				idle := vals[3]
				if len(vals) > 4 {
					idle += vals[4]
				}
				s.BusyTicks = s.TotalTicks - idle
				s.SoftIRQTicks = vals[6]
			}
		}
	}
	if iface != "" {
		b, err := os.ReadFile("/proc/net/dev")
		if err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if !strings.Contains(line, ":") {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				if strings.TrimSpace(parts[0]) != iface {
					continue
				}
				f := strings.Fields(parts[1])
				if len(f) >= 10 {
					s.NetRXBytes, _ = strconv.ParseUint(f[0], 10, 64)
					s.NetRXPackets, _ = strconv.ParseUint(f[1], 10, 64)
					s.NetTXBytes, _ = strconv.ParseUint(f[8], 10, 64)
					s.NetTXPackets, _ = strconv.ParseUint(f[9], 10, 64)
				}
			}
		}
	}
	return s, nil
}

func enrichProc(r *Result, before, after procSample, elapsed float64) {
	if elapsed <= 0 {
		return
	}
	total := after.TotalTicks - before.TotalTicks
	if total > 0 {
		busy := after.BusyTicks - before.BusyTicks
		soft := after.SoftIRQTicks - before.SoftIRQTicks
		r.HostBusyPercent = float64(busy) / float64(total) * 100
		r.HostSoftIRQPercent = float64(soft) / float64(total) * 100
		if after.ProcTicks >= before.ProcTicks && r.Requests > 0 {
			proc := after.ProcTicks - before.ProcTicks
			r.ProcessCPUPercent = float64(proc) / float64(total) * float64(runtime.NumCPU()) * 100
			r.ProcessCPUCores = r.ProcessCPUPercent / 100
			if r.RPS > 0 {
				r.CPUUSPerRequest = r.ProcessCPUCores * 1e6 / r.RPS
			}
		}
	}
	r.RSSStartMiB = float64(before.RSSKiB) / 1024
	r.RSSEndMiB = float64(after.RSSKiB) / 1024
	r.NetRXPPS = float64(after.NetRXPackets-before.NetRXPackets) / elapsed
	r.NetTXPPS = float64(after.NetTXPackets-before.NetTXPackets) / elapsed
	r.NetRXMbps = float64(after.NetRXBytes-before.NetRXBytes) * 8 / elapsed / 1e6
	r.NetTXMbps = float64(after.NetTXBytes-before.NetTXBytes) * 8 / elapsed / 1e6
}

func platformDetails() (string, string) {
	model := ""
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.ToLower(line), "model name") {
				if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
					model = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}
	kernel := ""
	if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		kernel = strings.TrimSpace(string(b))
	}
	return model, kernel
}
