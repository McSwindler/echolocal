//go:build !linux

package parts

func setLabel(string) error { return nil }
