package assets

import _ "embed"

// What an install writes to the device, other than echod itself. These are committed rather than
// staged, so every build carries them: a couple of kilobytes together.
//
// The boot image lives in internal/host/bootimg, which publishes it as a release asset and fetches
// it at install time against a compiled-in hash. Nine megabytes per board is more than echoctl
// should carry.

//go:embed patch/default.prop
var defaultProp []byte

//go:embed patch/fstab.mt8163
var fstab []byte

// The root filesystem lives on the system partition on this device, so these go there.
func DefaultProp() []byte { return defaultProp }
func Fstab() []byte       { return fstab }
