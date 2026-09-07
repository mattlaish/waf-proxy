package main

import "sync"

const reverseProxyCopyBufferSize = 32 << 10

type reverseProxyCopyBuffer [reverseProxyCopyBufferSize]byte

// proxyBufferPool implements httputil.BufferPool using pointers to fixed-size
// arrays. Keeping pointers in sync.Pool avoids allocating a slice header on
// each Put while still presenting []byte to ReverseProxy.
type proxyBufferPool struct {
	pool sync.Pool
}

func newProxyBufferPool() *proxyBufferPool {
	p := &proxyBufferPool{}
	p.pool.New = func() any { return new(reverseProxyCopyBuffer) }
	return p
}

func (p *proxyBufferPool) Get() []byte {
	buf := p.pool.Get().(*reverseProxyCopyBuffer)
	return buf[:]
}

func (p *proxyBufferPool) Put(buf []byte) {
	// Do not retain unexpected application buffers. A subslice with the same
	// capacity is safe to expand back to the fixed ReverseProxy copy size.
	if cap(buf) != reverseProxyCopyBufferSize {
		return
	}
	full := buf[:reverseProxyCopyBufferSize]
	p.pool.Put((*reverseProxyCopyBuffer)(full))
}

var reverseProxyCopyBuffers = newProxyBufferPool()
