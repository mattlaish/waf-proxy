//go:build !linux

package hsm

func validateModuleOwnership(string, []string) error { return nil }
