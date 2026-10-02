//go:build !board_checkers && !board_cronos && !board_rook

package config

type words struct{}

var say words

func (words) T(id string) string { return id }
