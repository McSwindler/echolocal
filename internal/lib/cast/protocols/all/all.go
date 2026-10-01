// Package all pulls in every cast protocol, each defining itself on import.
package all

import (
	_ "github.com/ygelfand/echolocal/internal/lib/cast/protocols/unsupported"
	_ "github.com/ygelfand/echolocal/internal/lib/cast/protocols/youtube"
)
