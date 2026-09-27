package sysimg

// Layout is everything about one board's system partition that an install has to know: where its
// dm-verity metadata sits, which files are replaced wholesale, and where the four bytes are that
// stop its adbd dropping privileges.
//
// Every number in one was measured against a particular system image. Sharing a layout between
// boards would mean writing bytes at an offset measured on a different filesystem, which is why
// these belong to a board rather than to the package.
//
// A zero Layout is a board that needs none of it. Where the root filesystem lives in the boot
// ramdisk rather than on the system partition, the fstab, the properties and adbd all ride in the
// image we write, and an fstab with no verify flag leaves nothing for dm-verity to check.
type Layout struct {
	// VerityOffset is where the system partition's dm-verity metadata starts.
	VerityOffset int64

	// Patches are the files replaced whole, each with the hash it has to be.
	Patches []Patch

	// Adbd is the in-place patch to the adb daemon.
	Adbd BinPatch

	// OTA is the property that stops Fire OS taking an update over this install.
	OTAKey, OTAValue string

	// Props are properties set in PropsPath.
	PropsPath string
	Props     map[string]string

	// Gate are the init services stopped from starting.
	Gate []string

	// ServiceRC is the rc file replaced with the definition that starts echod.
	ServiceRC string
}

// Empty reports whether the system partition needs nothing written to it.
func (l Layout) Empty() bool {
	return l.VerityOffset == 0 && len(l.Patches) == 0 && l.Adbd.Path == "" &&
		len(l.Props) == 0 && len(l.Gate) == 0 && l.ServiceRC == ""
}

// BinPatch is bytes overwritten in place at a fixed offset in a file. Patching rather than replacing
// the binary means a build this was not derived from is refused instead of overwritten.
type BinPatch struct {
	Path          string
	Offset        int64
	Before, After []byte
}

// Biscuit is the 2nd-generation Echo Dot's system partition.
var Biscuit = Layout{
	VerityOffset: VerityOffset,
	Patches:      []Patch{DefaultProp, Fstab},
	Adbd:         Adbd,
	OTAKey:       OTAKey,
	OTAValue:     OTAValue,
}

// Checkers and Cronos keep their root filesystem in the boot ramdisk, so nothing is written to the
// system partition and no fstab asks for dm-verity.
var (
	Checkers = Layout{
		OTAKey:   OTAKey,
		OTAValue: OTAValue,
	}
	Cronos = Layout{
		OTAKey:   OTAKey,
		OTAValue: OTAValue,
	}
)

// Rook is the 1st-generation Echo Spot's system partition, which holds LineageOS.
var Rook = Layout{
	PropsPath: "default.prop",
	Props: map[string]string{
		"ro.secure":              "0",
		"ro.adb.secure":          "0",
		"persist.sys.usb.config": "adb",
	},

	Gate:      []string{"surfaceflinger", "audioserver"},
	ServiceRC: "system/etc/init/hw/init.zygote32.rc",
}
