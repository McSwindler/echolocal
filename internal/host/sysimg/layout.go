package sysimg

// Layout is everything about one board's system partition that an install has to know: where its
// dm-verity metadata sits, which files are replaced wholesale, and where the four bytes are that
// stop its adbd dropping privileges.
//
// Every number in one was measured against a particular system image. Sharing a layout between
// boards would mean writing bytes at an offset measured on a different filesystem, which is why
// these belong to a board rather than to the package.
type Layout struct {
	// VerityOffset is where the system partition's dm-verity metadata starts.
	VerityOffset int64

	// Patches are the files replaced whole, each with the hash it has to be.
	Patches []Patch

	// Adbd is the in-place patch to the adb daemon.
	Adbd BinPatch

	// OTA is the property that stops Fire OS taking an update over this install.
	OTAKey, OTAValue string
}

// BinPatch is bytes overwritten in place at a fixed offset in a file. Patching rather than replacing
// the binary means a build this was not derived from is refused instead of overwritten.
type BinPatch struct {
	Path          string
	Offset        int64
	Before, After []byte
}

// Biscuit is the 2nd-generation Echo Dot's system partition, the only one any of this was measured
// against.
var Biscuit = Layout{
	VerityOffset: VerityOffset,
	Patches:      []Patch{DefaultProp, Fstab},
	Adbd:         Adbd,
	OTAKey:       OTAKey,
	OTAValue:     OTAValue,
}
