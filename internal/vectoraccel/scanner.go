package vectoraccel

import "errors"

type compiledScanner interface {
	Scan([]byte) ([]int, error)
	Close() error
}
type scannerFactory interface {
	Available() bool
	Version() string
	Compile([]RuleSpec) (compiledScanner, error)
}

var ErrNativeUnavailable = errors.New("VectorScan native scanner unavailable")
