package theme

// The mark's own colours, sampled from the artwork: the blue is over half of what it draws in
// colour and the orange is the warm end of the same gradient.
//
// These are the device's identity rather than a theme. A screen that belongs to EchoLocal itself —
// starting up, asking to be set up — uses them whatever theme is chosen, so it looks like the same
// product every time.
var (
	Blue   = rgb(0x0c9cfc)
	Orange = rgb(0xfc8c0c)

	// Ink is the ground the mark was drawn against, and Paper its light counterpart.
	Ink   = rgb(0x0e0e12)
	Paper = rgb(0xf7f9fb)

	// Slate is for the words that support the ones being read.
	Slate = rgb(0x6b7b8c)

	// Mist is the ground a card sits on: paper taken a few steps toward the ink, so the card is
	// the brighter of the two and reads as being on top of it.
	Mist = rgb(0xe4ebf2)
)

// Brand is the palette for screens the device owns.
func Brand() Theme {
	return Theme{
		Name: "EchoLocal",
		Dark: false,

		Background: Mist,
		Surface:    Paper,

		Text:  Ink,
		Muted: Slate,

		Accent:  Blue,
		Accent2: Orange,

		Success: rgb(0x2e9e57),
		Warning: rgb(0xb8791a),
		Danger:  rgb(0xd13b3b),
	}
}
