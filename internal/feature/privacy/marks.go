package privacy

// Marks is what the physical controls are doing.
type Marks struct {
	MicMuted      bool
	CameraBlocked bool
}

// Showing reports whether there is anything to draw.
func (m Marks) Showing() bool { return m.MicMuted || m.CameraBlocked }
