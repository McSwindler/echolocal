//go:build board_checkers || board_cronos || board_rook

package config

import text "github.com/ygelfand/echolocal/internal/lib/say"

type words struct{}

var say words

func (words) T(id string) string { return text.T(id) }
