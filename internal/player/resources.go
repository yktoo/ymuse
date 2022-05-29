package player

import _ "embed"

//go:embed glade/mpd-info.glade
var mpdInfoGlade string

//go:embed glade/outputs.glade
var outputsGlade string

//go:embed glade/player.glade
var playerGlade string

//go:embed glade/prefs.glade
var prefsGlade string

//go:embed glade/shortcuts.glade
var shortcutsGlade string
