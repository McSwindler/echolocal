// Package config is everything the device is set to, and the one place that decides it.
//
// Reading takes no lock and carries no default:
//
//	c := config.Get()
//	c.Speaker.Volume
//
// Writing names the thing being changed, and persists it:
//
//	config.Set().Speaker().Volume(8)
//
// Each part of the device gets a file here holding its own struct, its defaults and its writer, so
// a setting and everything about it is in one place. Nothing distinguishes "never set" from "set to
// the default": loading starts from the defaults and lets the file write over what it mentions.
//
// The values the user can choose between are named here too, as closed sets of identifiers with a
// label — the identifier is written to the file and branched on, the label is what Home Assistant
// shows. Keeping both here means persistence does not import the subsystem a setting belongs to,
// and the subsystem does not import persistence.
package config

import (
	"fmt"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/defaults"
)

// Config is what the device is set to.
type Config struct {
	Device     Device     `json:"-"`
	Speaker    Speaker    `json:"speaker"`
	Volume     Volume     `json:"volume"`
	Microphone Microphone `json:"microphone"`
	Wake       Wake       `json:"wake"`
	Ring       Ring       `json:"ring"`
	Update     Update     `json:"update"`
	Bluetooth  Bluetooth  `json:"bluetooth"`
	Diag       Diag       `json:"diag"`
	Media      Media      `json:"media"`
	Sendspin   Sendspin   `json:"sendspin"`
	Screen     Screen     `json:"screen"`
	Clock      Clock      `json:"clock"`
	Visual     Visual     `json:"visual"`
	Idle       Idle       `json:"idle"`
	Cast       Cast       `json:"cast"`
	Camera     Camera     `json:"camera"`
	RTSP       RTSP       `json:"rtsp"`
	Network    Network    `json:"network"`
	API        API        `json:"api"`
}

// Defaults is a device nobody has set anything on.
//
// The values that are statements about a board rather than choices come from defaults, which has
// been told which board this is. Everything here is what the file is merged over, so a setting
// absent from the file is the board's answer and not a Dot's.
func Defaults() Config {
	d := defaults.Current()

	return Config{
		Speaker:    defaultSpeaker(),
		Volume:     defaultVolume(),
		Microphone: defaultMicrophone(d),
		Ring:       defaultRing(d),
		Update:     defaultUpdate(),
		Bluetooth:  defaultBluetooth(),
		Diag:       defaultDiag(d),
		Media:      defaultMedia(),
		Sendspin:   defaultSendspin(),
		Screen:     defaultScreen(),
		Clock:      defaultClock(),
		Visual:     defaultVisual(),
		Idle:       defaultIdle(),
		Cast:       defaultCast(),
		Camera:     defaultCamera(),
		RTSP:       defaultRTSP(),

		// Only the stop word. The slots are absent until something chooses one, and Slot fills in the
		// defaults for whichever have not been.
		Wake: Wake{Stop: defaultStop()},
	}
}

// Device is what echod was told at start-up rather than what anyone chose. It is read like
// everything else, and not written to the file: the next process is told again.
//
// Nothing here is readable until boot has called Started, so a component built during init must not
// reach for it.
type Device struct {
	Name string
	Addr string

	// Board is which model this is. Everything that presents the device to Home Assistant reads it
	// from here rather than detecting again, so there is one answer per process.
	Board board.Board
}

// Writer is what Set hands back: one method per part of the device, each with its own settings.
//
// Nothing here holds a lock. The leaf call does the whole thing — take the lock, change the value,
// write the file — and returns whether the file was written.
type Writer struct{ st *Store }

func (w Writer) Speaker() SpeakerWriter       { return SpeakerWriter(w) }
func (w Writer) Volume() VolumeWriter         { return VolumeWriter(w) }
func (w Writer) Microphone() MicrophoneWriter { return MicrophoneWriter(w) }
func (w Writer) Ring() RingWriter             { return RingWriter(w) }
func (w Writer) Update() UpdateWriter         { return UpdateWriter(w) }
func (w Writer) Bluetooth() BluetoothWriter   { return BluetoothWriter(w) }
func (w Writer) Diag() DiagWriter             { return DiagWriter(w) }
func (w Writer) Media() MediaWriter           { return MediaWriter(w) }
func (w Writer) Sendspin() SendspinWriter     { return SendspinWriter(w) }
func (w Writer) Screen() ScreenWriter         { return ScreenWriter(w) }
func (w Writer) Clock() ClockWriter           { return ClockWriter(w) }
func (w Writer) Visual() VisualWriter         { return VisualWriter(w) }
func (w Writer) Idle() IdleWriter             { return IdleWriter(w) }
func (w Writer) Cast() CastWriter             { return CastWriter(w) }
func (w Writer) Camera() CameraWriter         { return CameraWriter(w) }
func (w Writer) RTSP() RTSPWriter             { return RTSPWriter(w) }
func (w Writer) Network() NetworkWriter       { return NetworkWriter(w) }
func (w Writer) API() APIWriter               { return APIWriter(w) }

// Wake names one slot, since every wake word setting belongs to one.
func (w Writer) Wake(slot int) WakeWriter { return WakeWriter{st: w.st, slot: slot} }

// Stop is the stop word, which is not a slot.
func (w Writer) Stop() StopWriter { return StopWriter(w) }

func errSlot(n int) error { return fmt.Errorf("config: wake slot %d", n) }

// Labelled is a setting whose values name themselves. The entity layer binds any of these to a
// select without knowing which setting it is.
type Labelled interface{ Label() string }

// Labels is the list Home Assistant shows for a set of values, in the order given.
func Labels[T Labelled](values []T) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.Label())
	}
	return out
}

// ByLabel resolves what Home Assistant sent back to the value it names. A select speaks labels,
// everything else speaks values, and this is the one place the two meet.
func ByLabel[T Labelled](values []T, label string) (T, bool) {
	for _, v := range values {
		if v.Label() == label {
			return v, true
		}
	}
	var zero T
	return zero, false
}
