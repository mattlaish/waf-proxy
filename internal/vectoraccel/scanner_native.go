//go:build vectorscan && cgo

package vectoraccel

/*
#cgo pkg-config: libhs
#include <stdlib.h>
#include <stdint.h>
#include <hs/hs.h>
extern int goVectorMatch(unsigned int id, unsigned long long from, unsigned long long to, unsigned int flags, uintptr_t handle);
typedef struct vector_ctx { uintptr_t handle; } vector_ctx_t;
static vector_ctx_t *vector_ctx_new(uintptr_t handle) {
    vector_ctx_t *ctx = (vector_ctx_t *)malloc(sizeof(vector_ctx_t));
    if (ctx != NULL) ctx->handle = handle;
    return ctx;
}
static void vector_ctx_free(vector_ctx_t *ctx) { free(ctx); }
static int vector_match_cb(unsigned int id, unsigned long long from, unsigned long long to, unsigned int flags, void *opaque) {
    vector_ctx_t *ctx = (vector_ctx_t *)opaque;
    return goVectorMatch(id, from, to, flags, ctx->handle);
}
static hs_error_t vector_scan(hs_database_t *db, const char *data, unsigned int length, hs_scratch_t *scratch, vector_ctx_t *ctx) {
    return hs_scan(db, data, length, 0, scratch, vector_match_cb, ctx);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"runtime/cgo"
	"sync"
	"unsafe"
)

type nativeFactory struct{}

func (nativeFactory) Available() bool { return true }
func (nativeFactory) Version() string {
	p := C.hs_version()
	if p == nil {
		return "libhs"
	}
	return C.GoString(p)
}
func newNativeFactory() scannerFactory { return nativeFactory{} }

type nativeDB struct {
	mu      sync.RWMutex
	db      *C.hs_database_t
	scratch chan *C.hs_scratch_t
	closed  bool
}

func (nativeFactory) Compile(rules []RuleSpec) (compiledScanner, error) {
	if len(rules) == 0 {
		return nil, fmt.Errorf("empty pattern group")
	}
	exprs := make([]*C.char, len(rules))
	flags := make([]C.uint, len(rules))
	ids := make([]C.uint, len(rules))
	for i, r := range rules {
		exprs[i] = C.CString(r.Pattern)
		defer C.free(unsafe.Pointer(exprs[i]))
		flags[i] = C.HS_FLAG_DOTALL
		ids[i] = C.uint(r.ID)
	}
	var db *C.hs_database_t
	var ce *C.hs_compile_error_t
	rc := C.hs_compile_multi((**C.char)(unsafe.Pointer(&exprs[0])), (*C.uint)(unsafe.Pointer(&flags[0])), (*C.uint)(unsafe.Pointer(&ids[0])), C.uint(len(rules)), C.HS_MODE_BLOCK, nil, &db, &ce)
	if rc != C.HS_SUCCESS {
		msg, idx := "compile failed", -1
		if ce != nil {
			msg = C.GoString(ce.message)
			idx = int(ce.expression)
			C.hs_free_compile_error(ce)
		}
		return nil, fmt.Errorf("VectorScan compile expression %d: %s", idx, msg)
	}
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		n = 1
	}
	d := &nativeDB{db: db, scratch: make(chan *C.hs_scratch_t, n)}
	var base *C.hs_scratch_t
	if C.hs_alloc_scratch(db, &base) != C.HS_SUCCESS {
		C.hs_free_database(db)
		return nil, fmt.Errorf("VectorScan scratch allocation failed")
	}
	d.scratch <- base
	for i := 1; i < n; i++ {
		var s *C.hs_scratch_t
		if C.hs_clone_scratch(base, &s) == C.HS_SUCCESS {
			d.scratch <- s
		}
	}
	return d, nil
}

type matchCollector struct{ ids []int }

//export goVectorMatch
func goVectorMatch(id C.uint, from C.ulonglong, to C.ulonglong, flags C.uint, handle C.uintptr_t) C.int {
	h := cgo.Handle(handle)
	c := h.Value().(*matchCollector)
	c.ids = append(c.ids, int(id))
	return 0
}

func (d *nativeDB) Scan(b []byte) ([]int, error) {
	if len(b) == 0 {
		return nil, nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed || d.db == nil {
		return nil, fmt.Errorf("VectorScan database closed")
	}
	var s *C.hs_scratch_t
	select {
	case s = <-d.scratch:
	default:
		if C.hs_alloc_scratch(d.db, &s) != C.HS_SUCCESS {
			return nil, fmt.Errorf("VectorScan scratch allocation failed")
		}
	}
	pooled := false
	defer func() {
		select {
		case d.scratch <- s:
			pooled = true
		default:
		}
		if !pooled {
			C.hs_free_scratch(s)
		}
	}()
	c := &matchCollector{}
	h := cgo.NewHandle(c)
	defer h.Delete()
	ctx := C.vector_ctx_new(C.uintptr_t(h))
	if ctx == nil {
		return nil, fmt.Errorf("VectorScan callback context allocation failed")
	}
	defer C.vector_ctx_free(ctx)
	rc := C.vector_scan(d.db, (*C.char)(unsafe.Pointer(&b[0])), C.uint(len(b)), s, ctx)
	if rc != C.HS_SUCCESS {
		return nil, fmt.Errorf("VectorScan scan failed: %d", int(rc))
	}
	return c.ids, nil
}
func (d *nativeDB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	close(d.scratch)
	for s := range d.scratch {
		C.hs_free_scratch(s)
	}
	if d.db != nil {
		C.hs_free_database(d.db)
		d.db = nil
	}
	return nil
}
