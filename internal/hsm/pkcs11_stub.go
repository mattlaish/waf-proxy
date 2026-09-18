//go:build !linux || !cgo || !pkcs11

package hsm

import "errors"

func openPlatformPKCS11Module(string) (nativeModule, error) {
	return nil, errors.New("PKCS#11 provider requires Linux with CGO enabled")
}
func encodeNativeULongs(vals ...uint64) []byte {
	out := make([]byte, 8*len(vals))
	for i, v := range vals {
		for j := 0; j < 8; j++ {
			out[i*8+j] = byte(v >> (8 * j))
		}
	}
	return out
}
