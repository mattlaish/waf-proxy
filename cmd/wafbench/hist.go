package main

import "math/bits"

const histSubBuckets = 64
const histBuckets = 64 * histSubBuckets

type latencyHist [histBuckets]uint64

func (h *latencyHist) addNs(ns int64) {
	if ns < 1 {
		ns = 1
	}
	v := uint64(ns)
	exp := bits.Len64(v) - 1
	base := uint64(1) << exp
	frac := uint64(0)
	if base > 0 {
		frac = ((v - base) * histSubBuckets) / base
	}
	if frac >= histSubBuckets {
		frac = histSubBuckets - 1
	}
	idx := exp*histSubBuckets + int(frac)
	if idx >= len(h) {
		idx = len(h) - 1
	}
	h[idx]++
}

func (h *latencyHist) merge(other *latencyHist) {
	for i := range h {
		h[i] += other[i]
	}
}

func (h *latencyHist) total() uint64 {
	var n uint64
	for _, v := range h {
		n += v
	}
	return n
}

func (h *latencyHist) quantile(q float64) float64 {
	total := h.total()
	if total == 0 {
		return 0
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	target := uint64(float64(total-1)*q) + 1
	var seen uint64
	for idx, n := range h {
		seen += n
		if seen >= target {
			exp := idx / histSubBuckets
			sub := idx % histSubBuckets
			base := uint64(1) << exp
			low := base + (base*uint64(sub))/histSubBuckets
			return float64(low) / 1e6 // ns -> ms
		}
	}
	return 0
}
