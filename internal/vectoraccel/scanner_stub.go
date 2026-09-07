//go:build !vectorscan

package vectoraccel

type nativeFactory struct{}

func (nativeFactory) Available() bool                             { return false }
func (nativeFactory) Version() string                             { return "unavailable" }
func (nativeFactory) Compile([]RuleSpec) (compiledScanner, error) { return nil, ErrNativeUnavailable }
func newNativeFactory() scannerFactory                            { return nativeFactory{} }
