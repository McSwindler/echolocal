// Package assets holds what echoctl ships inside itself: the wake word models and the files an
// install writes onto the system partition.
//
// echod is not among them. It is built per board, so one embedded copy is the wrong one for some
// device; it is resolved from a manifest at install time instead.
package assets
