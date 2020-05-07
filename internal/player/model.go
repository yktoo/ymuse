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

package player

import (
	"github.com/yktoo/ymuse/internal/util"
	"path"
	"sort"
)

//noinspection GoSnakeCaseUsage
const (
	MTA_Artist = iota
	MTA_ArtistSort
	MTA_Album
	MTA_AlbumSort
	MTA_AlbumArtist
	MTA_AlbumArtistSort
	MTA_Disc
	MTA_Track
	MTA_Number
	MTA_Length
	MTA_Path
	MTA_Directory
	MTA_File
	MTA_Year
	MTA_Genre
	MTA_Name
	MTA_Composer
	MTA_Performer
	MTA_Conductor
	MTA_Work
	MTA_Grouping
	MTA_Comment
	MTA_Label
)

type MpdTrackAttribute struct {
	name      string                // Short display label for the attribute
	longName  string                // Display label for the attribute
	attrName  string                // Internal name of the corresponding MPD attribute
	numeric   bool                  // Whether the attribute's value is numeric
	width     int                   // Default width of the column displaying this attribute
	formatter func(v string) string // Optional function for formatting the value
}

// Known MPD attributes
var MpdTrackAttributes = map[int]MpdTrackAttribute{
	MTA_Artist:          {"Artist", "Artist", "Artist", false, 200, nil},
	MTA_ArtistSort:      {"Artist", "Artist (for sorting)", "Artistsort", false, 200, nil},
	MTA_Album:           {"Album", "Album", "Album", false, 200, nil},
	MTA_AlbumSort:       {"Album", "Album (for sorting)", "Albumsort", false, 200, nil},
	MTA_AlbumArtist:     {"Album artist", "Album artist", "Albumartist", false, 200, nil},
	MTA_AlbumArtistSort: {"Album artist", "Album artist (for sorting)", "Albumartistsort", false, 200, nil},
	MTA_Disc:            {"Disc", "Disc", "Disc", false, 50, nil},
	MTA_Track:           {"Track", "Track title", "Title", false, 200, nil},
	MTA_Number:          {"#", "Track number", "Track", true, 50, nil},
	MTA_Length:          {"Length", "Track length", "duration", true, 60, util.FormatSecondsStr},
	MTA_Path:            {"Path", "Directory and file name", "file", false, 200, nil},
	MTA_Directory:       {"Directory", "File path", "file", false, 200, path.Dir},
	MTA_File:            {"File", "File name", "file", false, 200, path.Base},
	MTA_Year:            {"Year", "Year", "Date", true, 50, nil},
	MTA_Genre:           {"Genre", "Genre", "Genre", false, 200, nil},
	MTA_Name:            {"Name", "Stream name", "Name", false, 200, nil},
	MTA_Composer:        {"Composer", "Composer", "Composer", false, 200, nil},
	MTA_Performer:       {"Performer", "Performer", "Performer", false, 200, nil},
	MTA_Conductor:       {"Conductor", "Conductor", "Conductor", false, 200, nil},
	MTA_Work:            {"Work", "Work", "Work", false, 200, nil},
	MTA_Grouping:        {"Grouping", "Grouping", "Grouping", false, 200, nil},
	MTA_Comment:         {"Comment", "Comment", "Comment", false, 200, nil},
	MTA_Label:           {"Label", "Label", "Label", false, 200, nil},
}

// Attribute IDs sorted in desired display order
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
