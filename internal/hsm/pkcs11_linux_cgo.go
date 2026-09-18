//go:build linux && cgo && pkcs11

package hsm

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef unsigned char CK_BYTE;
typedef CK_BYTE CK_BBOOL;
typedef unsigned long CK_ULONG;
typedef CK_ULONG CK_RV;
typedef CK_ULONG CK_SLOT_ID;
typedef CK_ULONG CK_SESSION_HANDLE;
typedef CK_ULONG CK_OBJECT_HANDLE;
typedef CK_ULONG CK_FLAGS;
typedef CK_ULONG CK_USER_TYPE;
typedef CK_ULONG CK_MECHANISM_TYPE;
typedef CK_ULONG CK_ATTRIBUTE_TYPE;
typedef void* CK_VOID_PTR;
typedef CK_BYTE* CK_BYTE_PTR;
typedef CK_ULONG* CK_ULONG_PTR;
typedef CK_SLOT_ID* CK_SLOT_ID_PTR;
typedef CK_SESSION_HANDLE* CK_SESSION_HANDLE_PTR;
typedef CK_OBJECT_HANDLE* CK_OBJECT_HANDLE_PTR;

typedef struct CK_VERSION { CK_BYTE major; CK_BYTE minor; } CK_VERSION;
typedef struct CK_TOKEN_INFO {
    CK_BYTE label[32]; CK_BYTE manufacturerID[32]; CK_BYTE model[16]; CK_BYTE serialNumber[16];
    CK_FLAGS flags; CK_ULONG ulMaxSessionCount; CK_ULONG ulSessionCount; CK_ULONG ulMaxRwSessionCount;
    CK_ULONG ulRwSessionCount; CK_ULONG ulMaxPinLen; CK_ULONG ulMinPinLen; CK_ULONG ulTotalPublicMemory;
    CK_ULONG ulFreePublicMemory; CK_ULONG ulTotalPrivateMemory; CK_ULONG ulFreePrivateMemory;
    CK_VERSION hardwareVersion; CK_VERSION firmwareVersion; CK_BYTE utcTime[16];
} CK_TOKEN_INFO;
typedef CK_TOKEN_INFO* CK_TOKEN_INFO_PTR;

typedef struct CK_MECHANISM { CK_MECHANISM_TYPE mechanism; CK_VOID_PTR pParameter; CK_ULONG ulParameterLen; } CK_MECHANISM;
typedef CK_MECHANISM* CK_MECHANISM_PTR;
typedef struct CK_ATTRIBUTE { CK_ATTRIBUTE_TYPE type; CK_VOID_PTR pValue; CK_ULONG ulValueLen; } CK_ATTRIBUTE;
typedef CK_ATTRIBUTE* CK_ATTRIBUTE_PTR;

typedef struct CK_FUNCTION_LIST {
    CK_VERSION version;
    void *C_Initialize; void *C_Finalize; void *C_GetInfo; void *C_GetFunctionList;
    void *C_GetSlotList; void *C_GetSlotInfo; void *C_GetTokenInfo; void *C_GetMechanismList; void *C_GetMechanismInfo;
    void *C_InitToken; void *C_InitPIN; void *C_SetPIN; void *C_OpenSession; void *C_CloseSession; void *C_CloseAllSessions;
    void *C_GetSessionInfo; void *C_GetOperationState; void *C_SetOperationState; void *C_Login; void *C_Logout;
    void *C_CreateObject; void *C_CopyObject; void *C_DestroyObject; void *C_GetObjectSize; void *C_GetAttributeValue; void *C_SetAttributeValue;
    void *C_FindObjectsInit; void *C_FindObjects; void *C_FindObjectsFinal;
    void *C_EncryptInit; void *C_Encrypt; void *C_EncryptUpdate; void *C_EncryptFinal;
    void *C_DecryptInit; void *C_Decrypt; void *C_DecryptUpdate; void *C_DecryptFinal;
    void *C_DigestInit; void *C_Digest; void *C_DigestUpdate; void *C_DigestKey; void *C_DigestFinal;
    void *C_SignInit; void *C_Sign; void *C_SignUpdate; void *C_SignFinal; void *C_SignRecoverInit; void *C_SignRecover;
    void *C_VerifyInit; void *C_Verify; void *C_VerifyUpdate; void *C_VerifyFinal; void *C_VerifyRecoverInit; void *C_VerifyRecover;
    void *C_DigestEncryptUpdate; void *C_DecryptDigestUpdate; void *C_SignEncryptUpdate; void *C_DecryptVerifyUpdate;
    void *C_GenerateKey; void *C_GenerateKeyPair; void *C_WrapKey; void *C_UnwrapKey; void *C_DeriveKey;
    void *C_SeedRandom; void *C_GenerateRandom; void *C_GetFunctionStatus; void *C_CancelFunction; void *C_WaitForSlotEvent;
} CK_FUNCTION_LIST;
typedef CK_FUNCTION_LIST* CK_FUNCTION_LIST_PTR;
typedef CK_FUNCTION_LIST_PTR* CK_FUNCTION_LIST_PTR_PTR;

typedef CK_RV (*FN_GetFunctionList)(CK_FUNCTION_LIST_PTR_PTR);
typedef CK_RV (*FN_Initialize)(CK_VOID_PTR);
typedef CK_RV (*FN_Finalize)(CK_VOID_PTR);
typedef CK_RV (*FN_GetSlotList)(CK_BBOOL, CK_SLOT_ID_PTR, CK_ULONG_PTR);
typedef CK_RV (*FN_GetTokenInfo)(CK_SLOT_ID, CK_TOKEN_INFO_PTR);
typedef CK_RV (*FN_OpenSession)(CK_SLOT_ID, CK_FLAGS, CK_VOID_PTR, CK_VOID_PTR, CK_SESSION_HANDLE_PTR);
typedef CK_RV (*FN_CloseSession)(CK_SESSION_HANDLE);
typedef CK_RV (*FN_Login)(CK_SESSION_HANDLE, CK_USER_TYPE, CK_BYTE_PTR, CK_ULONG);
typedef CK_RV (*FN_FindObjectsInit)(CK_SESSION_HANDLE, CK_ATTRIBUTE_PTR, CK_ULONG);
typedef CK_RV (*FN_FindObjects)(CK_SESSION_HANDLE, CK_OBJECT_HANDLE_PTR, CK_ULONG, CK_ULONG_PTR);
typedef CK_RV (*FN_FindObjectsFinal)(CK_SESSION_HANDLE);
typedef CK_RV (*FN_SignInit)(CK_SESSION_HANDLE, CK_MECHANISM_PTR, CK_OBJECT_HANDLE);
typedef CK_RV (*FN_Sign)(CK_SESSION_HANDLE, CK_BYTE_PTR, CK_ULONG, CK_BYTE_PTR, CK_ULONG_PTR);

#define CKR_OK 0x00000000UL
#define CKR_CRYPTOKI_ALREADY_INITIALIZED 0x00000191UL
#define CKR_USER_ALREADY_LOGGED_IN 0x00000100UL
#define CK_TRUE 1
#define CKF_SERIAL_SESSION 0x00000004UL
#define CKU_USER 1UL
#define CKO_PRIVATE_KEY 0x00000003UL
#define CKA_CLASS 0x00000000UL
#define CKA_LABEL 0x00000003UL
#define CKA_ID 0x00000102UL

typedef struct waf_p11_module { void *dl; CK_FUNCTION_LIST_PTR f; int finalize_owned; } waf_p11_module;

static waf_p11_module* waf_p11_open(const char *path, CK_RV *rv_out) {
    *rv_out = 0;
    void *dl = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (!dl) { *rv_out = (CK_RV)-1; return NULL; }
    FN_GetFunctionList get = (FN_GetFunctionList)dlsym(dl, "C_GetFunctionList");
    if (!get) { dlclose(dl); *rv_out=(CK_RV)-2; return NULL; }
    CK_FUNCTION_LIST_PTR f = NULL;
    CK_RV rv = get(&f);
    if (rv != CKR_OK || !f) { dlclose(dl); *rv_out=rv ? rv : (CK_RV)-3; return NULL; }
    FN_Initialize init = (FN_Initialize)f->C_Initialize;
    rv = init(NULL);
    int owned = 0;
    if (rv == CKR_OK) owned = 1;
    else if (rv != CKR_CRYPTOKI_ALREADY_INITIALIZED) { dlclose(dl); *rv_out=rv; return NULL; }
    waf_p11_module *m = (waf_p11_module*)calloc(1,sizeof(*m));
    if (!m) { if (owned) ((FN_Finalize)f->C_Finalize)(NULL); dlclose(dl); *rv_out=(CK_RV)-4; return NULL; }
    m->dl=dl; m->f=f; m->finalize_owned=owned; return m;
}
static CK_RV waf_p11_close(waf_p11_module *m) {
    if (!m) return CKR_OK;
    CK_RV rv=CKR_OK;
    if (m->finalize_owned && m->f && m->f->C_Finalize) rv=((FN_Finalize)m->f->C_Finalize)(NULL);
    if (m->dl) dlclose(m->dl); free(m); return rv;
}
static CK_RV waf_p11_slots(waf_p11_module *m, CK_SLOT_ID_PTR slots, CK_ULONG_PTR count) {
    return ((FN_GetSlotList)m->f->C_GetSlotList)(CK_TRUE, slots, count);
}
static CK_RV waf_p11_token_label(waf_p11_module *m, CK_SLOT_ID slot, CK_BYTE out[32]) {
    CK_TOKEN_INFO info; memset(&info,0,sizeof(info));
    CK_RV rv=((FN_GetTokenInfo)m->f->C_GetTokenInfo)(slot,&info);
    if (rv==CKR_OK) memcpy(out,info.label,32); return rv;
}
static CK_RV waf_p11_open_session(waf_p11_module *m, CK_SLOT_ID slot, CK_SESSION_HANDLE_PTR session) {
    return ((FN_OpenSession)m->f->C_OpenSession)(slot,CKF_SERIAL_SESSION,NULL,NULL,session);
}
static CK_RV waf_p11_close_session(waf_p11_module *m, CK_SESSION_HANDLE session) {
    return ((FN_CloseSession)m->f->C_CloseSession)(session);
}
static CK_RV waf_p11_login(waf_p11_module *m, CK_SESSION_HANDLE session, CK_BYTE_PTR pin, CK_ULONG pinlen) {
    CK_RV rv=((FN_Login)m->f->C_Login)(session,CKU_USER,pin,pinlen);
    return rv==CKR_USER_ALREADY_LOGGED_IN ? CKR_OK : rv;
}
static CK_RV waf_p11_find_key(waf_p11_module *m, CK_SESSION_HANDLE session, const char *label, CK_ULONG labellen, CK_BYTE_PTR id, CK_ULONG idlen, CK_OBJECT_HANDLE_PTR key) {
    CK_ULONG cls=CKO_PRIVATE_KEY; CK_ATTRIBUTE attrs[3]; CK_ULONG n=0;
    attrs[n++] = (CK_ATTRIBUTE){CKA_CLASS,&cls,sizeof(cls)};
    if (label && labellen) attrs[n++] = (CK_ATTRIBUTE){CKA_LABEL,(CK_VOID_PTR)label,labellen};
    if (id && idlen) attrs[n++] = (CK_ATTRIBUTE){CKA_ID,(CK_VOID_PTR)id,idlen};
    FN_FindObjectsInit fi=(FN_FindObjectsInit)m->f->C_FindObjectsInit; FN_FindObjects ff=(FN_FindObjects)m->f->C_FindObjects; FN_FindObjectsFinal fin=(FN_FindObjectsFinal)m->f->C_FindObjectsFinal;
    CK_RV rv=fi(session,attrs,n); if(rv!=CKR_OK) return rv;
    CK_OBJECT_HANDLE objs[2]={0,0}; CK_ULONG found=0; rv=ff(session,objs,2,&found); CK_RV frv=fin(session);
    if(rv!=CKR_OK) return rv; if(frv!=CKR_OK) return frv; if(found!=1) return (CK_RV)-10; *key=objs[0]; return CKR_OK;
}
static CK_RV waf_p11_sign(waf_p11_module *m, CK_SESSION_HANDLE session, CK_OBJECT_HANDLE key, CK_ULONG mech, CK_BYTE_PTR param, CK_ULONG paramlen, CK_BYTE_PTR data, CK_ULONG datalen, CK_BYTE_PTR out, CK_ULONG_PTR outlen) {
    CK_MECHANISM mechanism={mech,param,paramlen}; FN_SignInit si=(FN_SignInit)m->f->C_SignInit; FN_Sign sign=(FN_Sign)m->f->C_Sign;
    CK_RV rv=si(session,&mechanism,key); if(rv!=CKR_OK) return rv; return sign(session,data,datalen,out,outlen);
}
static CK_RV waf_p11_sign_len(waf_p11_module *m, CK_SESSION_HANDLE session, CK_OBJECT_HANDLE key, CK_ULONG mech, CK_BYTE_PTR param, CK_ULONG paramlen, CK_BYTE_PTR data, CK_ULONG datalen, CK_ULONG_PTR outlen) {
    return waf_p11_sign(m,session,key,mech,param,paramlen,data,datalen,NULL,outlen);
}
static CK_RV waf_p11_sign_after_len(waf_p11_module *m, CK_SESSION_HANDLE session, CK_BYTE_PTR data, CK_ULONG datalen, CK_BYTE_PTR out, CK_ULONG_PTR outlen) {
    FN_Sign sign=(FN_Sign)m->f->C_Sign;
    return sign(session,data,datalen,out,outlen);
}
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

type cgoPKCS11Module struct{ p *C.waf_p11_module }
type cgoPKCS11Session struct {
	m *cgoPKCS11Module
	h C.CK_SESSION_HANDLE
}

func openPlatformPKCS11Module(path string) (nativeModule, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var rv C.CK_RV
	p := C.waf_p11_open(cpath, &rv)
	if p == nil {
		return nil, fmt.Errorf("PKCS#11 module open failed (rv=0x%x)", uint64(rv))
	}
	return &cgoPKCS11Module{p: p}, nil
}
func (m *cgoPKCS11Module) Close() error {
	rv := C.waf_p11_close(m.p)
	m.p = nil
	if rv != 0 {
		return fmt.Errorf("PKCS#11 finalize failed (rv=0x%x)", uint64(rv))
	}
	return nil
}
func (m *cgoPKCS11Module) Slots() ([]uint64, error) {
	var n C.CK_ULONG
	if rv := C.waf_p11_slots(m.p, nil, &n); rv != 0 {
		return nil, fmt.Errorf("slot count failed (rv=0x%x)", uint64(rv))
	}
	if n == 0 {
		return nil, nil
	}
	slots := make([]C.CK_SLOT_ID, int(n))
	if rv := C.waf_p11_slots(m.p, &slots[0], &n); rv != 0 {
		return nil, fmt.Errorf("slot list failed (rv=0x%x)", uint64(rv))
	}
	out := make([]uint64, int(n))
	for i := range out {
		out[i] = uint64(slots[i])
	}
	return out, nil
}
func (m *cgoPKCS11Module) TokenLabel(slot uint64) (string, error) {
	var out [32]C.CK_BYTE
	rv := C.waf_p11_token_label(m.p, C.CK_SLOT_ID(slot), &out[0])
	if rv != 0 {
		return "", fmt.Errorf("token info failed (rv=0x%x)", uint64(rv))
	}
	b := C.GoBytes(unsafe.Pointer(&out[0]), 32)
	return strings.TrimRight(string(b), " \x00"), nil
}
func (m *cgoPKCS11Module) OpenSession(slot uint64) (nativeSession, error) {
	var h C.CK_SESSION_HANDLE
	rv := C.waf_p11_open_session(m.p, C.CK_SLOT_ID(slot), &h)
	if rv != 0 {
		return nil, fmt.Errorf("open session failed (rv=0x%x)", uint64(rv))
	}
	return &cgoPKCS11Session{m: m, h: h}, nil
}
func (s *cgoPKCS11Session) Close() error {
	if s.h == 0 {
		return nil
	}
	rv := C.waf_p11_close_session(s.m.p, s.h)
	s.h = 0
	if rv != 0 {
		return fmt.Errorf("close session failed (rv=0x%x)", uint64(rv))
	}
	return nil
}
func (s *cgoPKCS11Session) Login(pin []byte) error {
	if len(pin) == 0 {
		return errors.New("empty PIN")
	}
	rv := C.waf_p11_login(s.m.p, s.h, (*C.CK_BYTE)(unsafe.Pointer(&pin[0])), C.CK_ULONG(len(pin)))
	if rv != 0 {
		return fmt.Errorf("login failed (rv=0x%x)", uint64(rv))
	}
	return nil
}
func (s *cgoPKCS11Session) FindPrivateKey(label string, id []byte) (uint64, error) {
	var clabel *C.char
	if label != "" {
		clabel = C.CString(label)
		defer C.free(unsafe.Pointer(clabel))
	}
	var idp *C.CK_BYTE
	if len(id) > 0 {
		idp = (*C.CK_BYTE)(unsafe.Pointer(&id[0]))
	}
	var key C.CK_OBJECT_HANDLE
	rv := C.waf_p11_find_key(s.m.p, s.h, clabel, C.CK_ULONG(len(label)), idp, C.CK_ULONG(len(id)), &key)
	if rv != 0 {
		return 0, fmt.Errorf("private key lookup failed (rv=0x%x)", uint64(rv))
	}
	return uint64(key), nil
}
func (s *cgoPKCS11Session) Sign(key, mechanism uint64, param, data []byte) ([]byte, error) {
	var pp, dp *C.CK_BYTE
	if len(param) > 0 {
		pp = (*C.CK_BYTE)(unsafe.Pointer(&param[0]))
	}
	if len(data) > 0 {
		dp = (*C.CK_BYTE)(unsafe.Pointer(&data[0]))
	}
	var n C.CK_ULONG
	rv := C.waf_p11_sign_len(s.m.p, s.h, C.CK_OBJECT_HANDLE(key), C.CK_ULONG(mechanism), pp, C.CK_ULONG(len(param)), dp, C.CK_ULONG(len(data)), &n)
	if rv != 0 {
		return nil, fmt.Errorf("sign length failed (rv=0x%x)", uint64(rv))
	}
	if n == 0 || uint64(n) > 1<<20 {
		return nil, errors.New("invalid PKCS#11 signature length")
	}
	out := make([]byte, int(n))
	rv = C.waf_p11_sign_after_len(s.m.p, s.h, dp, C.CK_ULONG(len(data)), (*C.CK_BYTE)(unsafe.Pointer(&out[0])), &n)
	if rv != 0 {
		return nil, fmt.Errorf("sign failed (rv=0x%x)", uint64(rv))
	}
	return out[:int(n)], nil
}
func encodeNativeULongs(vals ...uint64) []byte {
	n := int(unsafe.Sizeof(C.CK_ULONG(0)))
	out := make([]byte, n*len(vals))
	for i, v := range vals {
		if n == 8 {
			if isLittleEndian() {
				binary.LittleEndian.PutUint64(out[i*n:], v)
			} else {
				binary.BigEndian.PutUint64(out[i*n:], v)
			}
		} else {
			if isLittleEndian() {
				binary.LittleEndian.PutUint32(out[i*n:], uint32(v))
			} else {
				binary.BigEndian.PutUint32(out[i*n:], uint32(v))
			}
		}
	}
	return out
}
func isLittleEndian() bool { var x uint16 = 1; return *(*byte)(unsafe.Pointer(&x)) == 1 }
