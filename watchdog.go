package main

// Optional hardware watchdog integration.
//
// IMPORTANT: software cannot make a powered-off box fail open — that is a
// hardware function (a fail-open/bypass NIC or an external bypass switch whose
// relay shorts the ports on power loss or loss of heartbeat). What this file
// provides is the SOFTWARE SIDE of that arrangement: it feeds a watchdog device
// while the WAF is healthy, and STOPS feeding it when the WAF is unhealthy or
// shutting down. The hardware then does the actual thing:
//
//   * A Linux kernel watchdog (/dev/watchdog): reboots the box when the feed
//     stops — recovers a hung process/host.
//   * A bypass-NIC watchdog: flips the relay to bypass (fail-open) when the
//     feed stops — traffic flows unfiltered past a dead WAF.
//
// Which behaviour you get depends on the hardware wired to the device, not on
// this code. Off by default. Enable only if you have such hardware, and set the
// feed interval well BELOW the hardware timeout or it will trip prematurely.

import (
	"os"
	"time"
)

// WAF_WATCHDOG_DEVICE (e.g. /dev/watchdog) enables the feeder when set.
// WAF_WATCHDOG_INTERVAL_SECONDS controls the feed cadence (default 10s).
func watchdogDevice() string { return os.Getenv("WAF_WATCHDOG_DEVICE") }

func watchdogInterval() time.Duration {
	if v := os.Getenv("WAF_WATCHDOG_INTERVAL_SECONDS"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return 10 * time.Second
}

// runWatchdog opens the watchdog device and feeds it on a timer while the WAF
// considers itself healthy. If the process crashes/hangs or marks itself
// draining, the feed stops and the hardware acts. Returns immediately (no-op)
// if no device is configured.
func (s *server) runWatchdog(stop <-chan struct{}) {
	dev := watchdogDevice()
	if dev == "" {
		return
	}
	f, err := os.OpenFile(dev, os.O_WRONLY, 0)
	if err != nil {
		// No watchdog hardware / permission: log and stay out of the way.
		s.log.Warn("watchdog device not available; feeder disabled", "device", dev, "err", err)
		return
	}
	s.log.Info("hardware watchdog feeder started", "device", dev, "interval", watchdogInterval().String())

	t := time.NewTicker(watchdogInterval())
	defer t.Stop()
	for {
		select {
		case <-stop:
			// Graceful stop: attempt the magic-close so the kernel watchdog is
			// disarmed on a clean shutdown (won't reboot). A bypass NIC that
			// interprets loss-of-feed as "go to bypass" will still do so.
			_, _ = f.Write([]byte("V")) // magic close (Linux watchdog API)
			_ = f.Close()
			return
		case <-t.C:
			// Only keep the watchdog fed while we're actually healthy. If the
			// runtime is gone or we're draining, stop feeding → hardware acts.
			if s.rt.Load() == nil || s.draining.Load() {
				s.log.Warn("watchdog: unhealthy — withholding keepalive so hardware can act")
				continue
			}
			if _, err := f.Write([]byte("w")); err != nil {
				s.log.Warn("watchdog write failed", "err", err)
			}
		}
	}
}

func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return 0, os.ErrInvalid
	}
	return n, nil
}
