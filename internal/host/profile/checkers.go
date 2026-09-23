package profile

import (
	"github.com/ygelfand/echolocal/internal/android/services"
	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/host/bootimg"
	"github.com/ygelfand/echolocal/internal/host/sysimg"
)

// checkers is the 1st-generation Echo Show 5.
var checkers = Profile{
	Board:  board.Checkers,
	Boot:   bootimg.Checkers,
	System: sysimg.Checkers,

	Disable: append([]string{
		"dha_service",
		"integrity_logger",
		"amazonfiled",
		"meshmgrservice",
		"inlocservice",
		"shblemeshd",
		"shchipd",
		"shlocalskillsd",
		"smarthomewifid",
		"securetime",
		"halo-hub-broker",
		"wifi_log_levels",
		"aal",
		"aal-metric",
		"touchlogger",
		"installd",
		"gatekeeperd",
		"keystore",
		"audioserver",
		"media",
		"mediacodec",
		"mediadrm",
		"mediaextractor",
		"cameraserver",
	}, services.Disabled...),
	Enable: services.Enabled,
}
