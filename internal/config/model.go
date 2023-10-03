/*
 *   Copyright 2020 Dmitry Kann
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"github.com/yktoo/ymuse/internal/util"
	"path"
	"sort"
)

// MPD's track attribute identifiers. These must precisely match the QueueListStore's columns declared in player.glade
const (
	MTAttrArtist = iota
	MTAttrArtistSort
	MTAttrAlbum
	MTAttrAlbumSort
	MTAttrAlbumArtist
	MTAttrAlbumArtistSort
	MTAttrDisc
	MTAttrTrack
	MTAttrNumber
	MTAttrLength
	MTAttrPath
	MTAttrDirectory
	MTAttrFile
	MTAttrYear
	MTAttrGenre
	MTAttrName
	MTAttrComposer
	MTAttrPerformer
	MTAttrConductor
	MTAttrWork
	MTAttrGrouping
	MTAttrComment
	MTAttrLabel
	MTAttrPos
	// List store's "artificial" columns used for rendering
	QueueColumnIcon
	QueueColumnFontWeight
	QueueColumnBgColor
	QueueColumnVisible
)

// MpdTrackAttribute describes an MPD's track attribute
type MpdTrackAttribute struct {
	Name            string                // Short display label for the attribute
	LongName        string                // Display label for the attribute
	AttrName        string                // Internal name of the corresponding MPD attribute
	Numeric         bool                  // Whether the attribute's value is numeric
	Searchable      bool                  // Whether the attribute is searchable
	Width           int                   // Default width of the column displaying this attribute
	XAlign          float64               // X alignment of the column displaying this attribute (0...1)
	Formatter       func(v string) string // Optional function for formatting the value
	FallbackAttrIDs []int                 // Optional references to the fallback attributes to use when there's no value, in the order of preference
}

// MpdTrackAttributes contains all known MPD's track attributes
var MpdTrackAttributes = map[int]MpdTrackAttribute{
	MTAttrArtist:          {"Artist", "Artist", "Artist", false, true, 200, 0, nil, nil},
	MTAttrArtistSort:      {"Artist", "Artist (for sorting)", "Artistsort", false, false, 200, 0, nil, nil},
	MTAttrAlbum:           {"Album", "Album", "Album", false, true, 200, 0, nil, nil},
	MTAttrAlbumSort:       {"Album", "Album (for sorting)", "Albumsort", false, false, 200, 0, nil, nil},
	MTAttrAlbumArtist:     {"Album artist", "Album artist", "Albumartist", false, true, 200, 0, nil, nil},
	MTAttrAlbumArtistSort: {"Album artist", "Album artist (for sorting)", "Albumartistsort", false, false, 200, 0, nil, nil},
	MTAttrDisc:            {"Disc", "Disc", "Disc", false, true, 50, 1, nil, nil},
	MTAttrTrack:           {"Track", "Track title", "Title", false, true, 200, 0, nil, []int{MTAttrName, MTAttrPath}},
	MTAttrNumber:          {"#", "Track number", "Track", true, true, 50, 1, nil, nil},
	MTAttrLength:          {"Length", "Track length", "duration", true, false, 60, 1, util.FormatSecondsStr, nil},
	MTAttrPath:            {"Path", "Directory and file name", "file", false, true, 200, 0, nil, nil},
	MTAttrDirectory:       {"Directory", "File path", "file", false, false, 200, 0, path.Dir, nil},
	MTAttrFile:            {"File", "File name", "file", false, false, 200, 0, path.Base, nil},
	MTAttrYear:            {"Year", "Year", "Date", true, true, 50, 1, nil, nil},
	MTAttrGenre:           {"Genre", "Genre", "Genre", false, true, 200, 0, nil, nil},
	MTAttrName:            {"Name", "Stream name", "Name", false, true, 200, 0, nil, nil},
	MTAttrComposer:        {"Composer", "Composer", "Composer", false, true, 200, 0, nil, nil},
	MTAttrPerformer:       {"Performer", "Performer", "Performer", false, true, 200, 0, nil, nil},
	MTAttrConductor:       {"Conductor", "Conductor", "Conductor", false, false, 200, 0, nil, nil},
	MTAttrWork:            {"Work", "Work", "Work", false, false, 200, 0, nil, nil},
	MTAttrGrouping:        {"Grouping", "Grouping", "Grouping", false, false, 200, 0, nil, nil},
	MTAttrComment:         {"Comment", "Comment", "Comment", false, true, 200, 0, nil, nil},
	MTAttrLabel:           {"Label", "Label", "Label", false, true, 200, 0, nil, nil},
	MTAttrPos:             {"Pos", "Position", "Pos", true, false, 0, 1, nil, nil},
}

// MpdTrackAttributeIds stores attribute IDs sorted in desired display order
var MpdTrackAttributeIds []int

func init() {
	// Fill in and sort MpdTrackAttributeIds
	MpdTrackAttributeIds = make([]int, len(MpdTrackAttributes))
	i := 0
	for id := range MpdTrackAttributes {
		MpdTrackAttributeIds[i] = id
		i++
	}
	sort.Ints(MpdTrackAttributeIds)
}
