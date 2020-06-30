/*
 * Copyright 2020 Dmitry Kann
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
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/gotk3/gotk3/glib"
	"github.com/yktoo/ymuse/internal/config"
	"github.com/yktoo/ymuse/internal/util"
	"path"
	"strings"
)

// LibraryPathElement represents one element in the library path
type LibraryPathElement interface {
	Icon() string                // Icon returns name of the icon for the element
	Label() string               // Label returns display label text for the element
	IsFolder() bool              // IsFolder denotes whether the element can be entered into (eg. by double-clicking it)
	IsPlayable() bool            // IsPlayable denotes whether the element can be played (added to the queue)
	Marshal() string             // Serialises the element into a string
	Unmarshal(data string) error // Deserialises the element from a string
	Prefix() string              // Returns a string prefix for marshalling
}

// URIHolder represents an object that possesses a URI
type URIHolder interface {
	URI() string
}

// AttributeHolder represents an object that possesses an MPD attribute and the corresponding value
type AttributeHolder interface {
	AttributeID() int       // Attribute's ID
	AttributeValue() string // Attribute's value
}

// AttributeHolderParent represents an object that can be a parent for AttributeHolder
type AttributeHolderParent interface {
	ChildAttributeID() int                    // Child attribute's ID
	NewChild(value string) LibraryPathElement // Instantiates and returns a new child element with the given value
}

// DetailsHolder represents an object that can provide additional details
type DetailsHolder interface {
	Details() string
}

// PlaylistHolder represents an object that references a playlist
type PlaylistHolder interface {
	PlaylistName() string
}

var elementConstructors = map[string]func() LibraryPathElement{
	"lvlup":      NewLevelUpLibElement,
	"filesystem": NewFilesystemLibElement,
	"dir":        NewDirLibElement,
	"file":       NewFileLibElement,
	"playlists":  NewPlaylistsLibElement,
	"playlist":   NewPlaylistLibElement,
	"genres":     NewGenresLibElement,
	"genre":      NewGenreLibElement,
	"artists":    NewArtistsLibElement,
	"artist":     NewArtistLibElement,
	"albums":     NewAlbumsLibElement,
	"album":      NewAlbumLibElement,
	"track":      NewTrackLibElement,
}

const (
	pathFieldSeparator   = "\u0001"
	pathElementSeparator = "\u0002"
)

// UnmarshalLibPathElement instantiates and initialises a library path element from the given serialised string form
func UnmarshalLibPathElement(data string) (LibraryPathElement, error) {
	// Extract type prefix
	i := strings.Index(data, pathFieldSeparator)
	if i < 0 {
		return nil, fmt.Errorf("failed to unmarshal library path element: missing prefix")
	}
	prefix := data[0:i]

	// If the prefix is known, instantiate and unmarshal the element
	if constructor, ok := elementConstructors[prefix]; ok {
		element := constructor()
		if err := element.Unmarshal(data[i+1:]); err != nil {
			return nil, err
		}
		return element, nil
	}

	// Prefix unknown
	return nil, fmt.Errorf("failed to unmarshal library path element: unknown prefix '%v'", prefix)
}

// MarshalLibPathElement serialises a library path element into string form
func MarshalLibPathElement(e LibraryPathElement) string {
	return e.Prefix() + pathFieldSeparator + e.Marshal()
}

// AttrsToElements converts the provided list of MPD Attributes instances into a slice of elements, stripping the
// (optional) URI prefix
func AttrsToElements(attrs []mpd.Attrs, uriPrefix string) []LibraryPathElement {
	result := make([]LibraryPathElement, 0, len(attrs))
	for _, a := range attrs {
		if dir, ok := a["directory"]; ok {
			result = append(result, &DirLibElement{
				uri:   dir,
				title: strings.TrimPrefix(dir, uriPrefix),
			})

		} else if file, ok := a["file"]; ok {
			result = append(result, &FileLibElement{
				uri:    file,
				title:  strings.TrimPrefix(file, uriPrefix),
				length: util.ParseFloatDef(a["duration"], 0.0),
			})

		} else if playlist, ok := a["playlist"]; ok {
			result = append(result, &PlaylistLibElement{
				name: strings.TrimPrefix(playlist, uriPrefix),
			})

		} else {
			continue
		}
	}
	return result
}

//----------------------------------------------------------------------------------------------------------------------
// LibraryPath
//----------------------------------------------------------------------------------------------------------------------

// LibraryPath holds a series of LibraryPathElement's
type LibraryPath struct {
	elements  []LibraryPathElement // Internal list of elements
	onChanged func()               // On path change callback
}

func NewLibraryPath(onChanged func()) *LibraryPath {
	return &LibraryPath{onChanged: onChanged}
}

// Append extends the current path with the given element
func (p *LibraryPath) Append(e LibraryPathElement) {
	p.elements = append(p.elements, e)
	p.onChanged()
}

// AsFilter converts the path (with optional additions) into a slice of arguments for MPD's filter function
func (p *LibraryPath) AsFilter(extraElements ...LibraryPathElement) (result []string) {
	// Iterate all elements, including extras
	for _, e := range append(p.elements, extraElements...) {
		// Select only those associated with attributes
		if ah, oka := e.(AttributeHolder); oka {
			// For each element, add two elements to the slice: the name and the value
			result = append(
				result,
				config.MpdTrackAttributes[ah.AttributeID()].AttrName,
				ah.AttributeValue())
		}
	}
	return
}

// ElementAt returns the element at the given index, or nil if no such element exists
func (p *LibraryPath) ElementAt(index int) LibraryPathElement {
	if index >= 0 && index < len(p.elements) {
		return p.elements[index]
	}
	return nil
}

// Elements returns the elements slice
func (p *LibraryPath) Elements() []LibraryPathElement {
	return p.elements
}

// IsRoot returns whether the current path represents root
func (p *LibraryPath) IsRoot() bool {
	return len(p.elements) == 0
}

// Last returns the last element in the current path, or nil if the path is empty
func (p *LibraryPath) Last() LibraryPathElement {
	if i := len(p.elements); i > 0 {
		return p.elements[i-1]
	}
	return nil
}

// LevelUp shifts the current path one level up (by dropping the last element)
func (p *LibraryPath) LevelUp() {
	if i := len(p.elements); i > 0 {
		p.elements = p.elements[:i-1]
		p.onChanged()
	}
}

// Marshal serialises the current path as a string
func (p *LibraryPath) Marshal() string {
	s := ""
	for _, e := range p.elements {
		if s != "" {
			s += pathElementSeparator
		}
		s += MarshalLibPathElement(e)
	}
	return s
}

// SetLength limits the length of the path at the given figure
func (p *LibraryPath) SetLength(length int) {
	if length > len(p.elements) {
		return
	}
	p.elements = p.elements[:length]
	p.onChanged()
}

// Unmarshal deserialises the path from a string
func (p *LibraryPath) Unmarshal(s string) error {
	// Iterate tab-separate serialised elements
	var elements []LibraryPathElement
	for _, s := range strings.Split(s, pathElementSeparator) {
		element, err := UnmarshalLibPathElement(s)
		if err != nil {
			return err
		}
		elements = append(elements, element)
	}

	// Succeeded
	p.elements = elements
	p.onChanged()
	return nil
}

//----------------------------------------------------------------------------------------------------------------------
// BaseAttrHolder: base (incomplete) implementation of AttributeHolder lacking NewChild()
//----------------------------------------------------------------------------------------------------------------------

type BaseAttrHolder struct {
	attrID    int
	attrValue string
}

func (h *BaseAttrHolder) AttributeID() int {
	return h.attrID
}

func (h *BaseAttrHolder) AttributeValue() string {
	return h.attrValue
}

//----------------------------------------------------------------------------------------------------------------------
// LevelUpLibElement - a LibraryPathElement that looks as a ".." and is used to navigate to the parent
//----------------------------------------------------------------------------------------------------------------------

type LevelUpLibElement struct{}

func NewLevelUpLibElement() LibraryPathElement {
	return &LevelUpLibElement{}
}

func (e *LevelUpLibElement) Icon() string {
	return "ymuse-level-up-symbolic"
}

func (e *LevelUpLibElement) Label() string {
	return ""
}

func (e *LevelUpLibElement) IsFolder() bool {
	return false
}

func (e *LevelUpLibElement) IsPlayable() bool {
	return false
}

func (e *LevelUpLibElement) Prefix() string {
	return "lvlup"
}

func (e *LevelUpLibElement) Marshal() string {
	return ""
}

func (e *LevelUpLibElement) Unmarshal(string) error {
	return nil
}

//----------------------------------------------------------------------------------------------------------------------
// FilesystemLibElement
//----------------------------------------------------------------------------------------------------------------------

type FilesystemLibElement struct{}

func NewFilesystemLibElement() LibraryPathElement {
	return &FilesystemLibElement{}
}

func (e *FilesystemLibElement) Icon() string {
	return "drive-harddisk"
}

func (e *FilesystemLibElement) Label() string {
	return glib.Local("Files")
}

func (e *FilesystemLibElement) IsFolder() bool {
	return true
}

func (e *FilesystemLibElement) IsPlayable() bool {
	return false
}

func (e *FilesystemLibElement) Prefix() string {
	return "filesystem"
}

func (e *FilesystemLibElement) Marshal() string {
	return ""
}

func (e *FilesystemLibElement) Unmarshal(string) error {
	return nil
}

func (e *FilesystemLibElement) URI() string {
	return ""
}

//----------------------------------------------------------------------------------------------------------------------
// DirLibElement
//----------------------------------------------------------------------------------------------------------------------

type DirLibElement struct {
	uri   string // URI of the directory
	title string // Title of the directory
}

func NewDirLibElement() LibraryPathElement {
	return &DirLibElement{}
}

func (e *DirLibElement) Icon() string {
	return "folder"
}

func (e *DirLibElement) Label() string {
	return path.Base(e.uri)
}

func (e *DirLibElement) IsFolder() bool {
	return true
}

func (e *DirLibElement) IsPlayable() bool {
	return true
}

func (e *DirLibElement) Prefix() string {
	return "dir"
}

func (e *DirLibElement) Marshal() string {
	return e.uri
}

func (e *DirLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal DirLibElement: want 1 field, got %d", len(fields))
	}
	e.uri = fields[0]
	return nil
}

func (e *DirLibElement) URI() string {
	return e.uri
}

//----------------------------------------------------------------------------------------------------------------------
// FileLibElement
//----------------------------------------------------------------------------------------------------------------------

type FileLibElement struct {
	uri    string  // URI of the file
	title  string  // Title of the track
	length float64 // Length of the track in seconds
}

func NewFileLibElement() LibraryPathElement {
	return &FileLibElement{}
}

func (e *FileLibElement) Icon() string {
	return "ymuse-audio-file"
}

func (e *FileLibElement) Label() string {
	return e.title
}

func (e *FileLibElement) IsFolder() bool {
	return false
}

func (e *FileLibElement) IsPlayable() bool {
	return true
}

func (e *FileLibElement) Prefix() string {
	return "file"
}

func (e *FileLibElement) Marshal() string {
	return e.uri + pathFieldSeparator + e.title
}

func (e *FileLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 2 {
		return fmt.Errorf("failed to unmarshal FileLibElement: want 2 fields, got %d", len(fields))
	}
	e.uri = fields[0]
	e.title = fields[1]
	return nil
}

func (e *FileLibElement) URI() string {
	return e.uri
}

func (e *FileLibElement) Details() string {
	if e.length > 0 {
		return util.FormatSeconds(e.length)
	}
	return ""
}

//----------------------------------------------------------------------------------------------------------------------
// PlaylistsLibElement
//----------------------------------------------------------------------------------------------------------------------

type PlaylistsLibElement struct{}

func NewPlaylistsLibElement() LibraryPathElement {
	return &PlaylistsLibElement{}
}

func (e *PlaylistsLibElement) Icon() string {
	return "ymuse-playlists"
}

func (e *PlaylistsLibElement) Label() string {
	return glib.Local("Playlists")
}

func (e *PlaylistsLibElement) IsFolder() bool {
	return true
}

func (e *PlaylistsLibElement) IsPlayable() bool {
	return false
}

func (e *PlaylistsLibElement) Prefix() string {
	return "playlists"
}

func (e *PlaylistsLibElement) Marshal() string {
	return ""
}

func (e *PlaylistsLibElement) Unmarshal(string) error {
	return nil
}

func (e *PlaylistsLibElement) NewChild(name string) LibraryPathElement {
	c := NewPlaylistLibElement()
	c.(*PlaylistLibElement).name = name
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// PlaylistLibElement
//----------------------------------------------------------------------------------------------------------------------

type PlaylistLibElement struct {
	name string // Playlist name
}

func NewPlaylistLibElement() LibraryPathElement {
	return &PlaylistLibElement{}
}

func (e *PlaylistLibElement) Icon() string {
	return "ymuse-playlist"
}

func (e *PlaylistLibElement) Label() string {
	return e.name
}

func (e *PlaylistLibElement) IsFolder() bool {
	return false
}

func (e *PlaylistLibElement) IsPlayable() bool {
	return true
}

func (e *PlaylistLibElement) Prefix() string {
	return "playlist"
}

func (e *PlaylistLibElement) Marshal() string {
	return e.name
}

func (e *PlaylistLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal PlaylistLibElement: want 1 fields, got %d", len(fields))
	}
	e.name = fields[0]
	return nil
}

func (e *PlaylistLibElement) PlaylistName() string {
	return e.name
}

//----------------------------------------------------------------------------------------------------------------------
// GenresLibElement
//----------------------------------------------------------------------------------------------------------------------

type GenresLibElement struct{}

func NewGenresLibElement() LibraryPathElement {
	return &GenresLibElement{}
}

func (e *GenresLibElement) Icon() string {
	return "ymuse-genres"
}

func (e *GenresLibElement) Label() string {
	return glib.Local("Genres")
}

func (e *GenresLibElement) IsFolder() bool {
	return true
}

func (e *GenresLibElement) IsPlayable() bool {
	return false
}

func (e *GenresLibElement) Prefix() string {
	return "genres"
}

func (e *GenresLibElement) Marshal() string {
	return ""
}

func (e *GenresLibElement) Unmarshal(string) error {
	return nil
}

func (e *GenresLibElement) ChildAttributeID() int {
	return config.MTAttrGenre
}

func (e *GenresLibElement) NewChild(value string) LibraryPathElement {
	c := NewGenreLibElement()
	c.(*GenreLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// GenreLibElement
//----------------------------------------------------------------------------------------------------------------------

type GenreLibElement struct {
	BaseAttrHolder
}

func NewGenreLibElement() LibraryPathElement {
	return &GenreLibElement{BaseAttrHolder{attrID: config.MTAttrGenre}}
}

func (e *GenreLibElement) Icon() string {
	return "ymuse-genre"
}

func (e *GenreLibElement) Label() string {
	if e.attrValue == "" {
		return glib.Local("(unknown)")
	}
	return e.attrValue
}

func (e *GenreLibElement) IsFolder() bool {
	return true
}

func (e *GenreLibElement) IsPlayable() bool {
	return true
}

func (e *GenreLibElement) Prefix() string {
	return "genre"
}

func (e *GenreLibElement) Marshal() string {
	return e.attrValue
}

func (e *GenreLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal GenreLibElement: want 1 field, got %d", len(fields))
	}
	e.attrValue = fields[0]
	return nil
}

func (e *GenreLibElement) ChildAttributeID() int {
	return config.MTAttrArtist
}

func (e *GenreLibElement) NewChild(value string) LibraryPathElement {
	c := NewArtistLibElement()
	c.(*ArtistLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// ArtistsLibElement
//----------------------------------------------------------------------------------------------------------------------

type ArtistsLibElement struct{}

func NewArtistsLibElement() LibraryPathElement {
	return &ArtistsLibElement{}
}

func (e *ArtistsLibElement) Icon() string {
	return "ymuse-artists"
}

func (e *ArtistsLibElement) Label() string {
	return glib.Local("Artists")
}

func (e *ArtistsLibElement) IsFolder() bool {
	return true
}

func (e *ArtistsLibElement) IsPlayable() bool {
	return false
}

func (e *ArtistsLibElement) Prefix() string {
	return "artists"
}

func (e *ArtistsLibElement) Marshal() string {
	return ""
}

func (e *ArtistsLibElement) Unmarshal(string) error {
	return nil
}

func (e *ArtistsLibElement) ChildAttributeID() int {
	return config.MTAttrArtist
}

func (e *ArtistsLibElement) NewChild(value string) LibraryPathElement {
	c := NewArtistLibElement()
	c.(*ArtistLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// ArtistLibElement
//----------------------------------------------------------------------------------------------------------------------

type ArtistLibElement struct {
	BaseAttrHolder
}

func NewArtistLibElement() LibraryPathElement {
	return &ArtistLibElement{BaseAttrHolder{attrID: config.MTAttrArtist}}
}

func (e *ArtistLibElement) Icon() string {
	return "ymuse-artist"
}

func (e *ArtistLibElement) Label() string {
	if e.attrValue == "" {
		return glib.Local("(unknown)")
	}
	return e.attrValue
}

func (e *ArtistLibElement) IsFolder() bool {
	return true
}

func (e *ArtistLibElement) IsPlayable() bool {
	return true
}

func (e *ArtistLibElement) Prefix() string {
	return "artist"
}

func (e *ArtistLibElement) Marshal() string {
	return e.attrValue
}

func (e *ArtistLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal ArtistLibElement: want 1 field, got %d", len(fields))
	}
	e.attrValue = fields[0]
	return nil
}

func (e *ArtistLibElement) ChildAttributeID() int {
	return config.MTAttrAlbum
}

func (e *ArtistLibElement) NewChild(value string) LibraryPathElement {
	c := NewAlbumLibElement()
	c.(*AlbumLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// AlbumsLibElement
//----------------------------------------------------------------------------------------------------------------------

type AlbumsLibElement struct{}

func NewAlbumsLibElement() LibraryPathElement {
	return &AlbumsLibElement{}
}

func (e *AlbumsLibElement) Icon() string {
	return "ymuse-albums"
}

func (e *AlbumsLibElement) Label() string {
	return glib.Local("Albums")
}

func (e *AlbumsLibElement) IsFolder() bool {
	return true
}

func (e *AlbumsLibElement) IsPlayable() bool {
	return false
}

func (e *AlbumsLibElement) Prefix() string {
	return "albums"
}

func (e *AlbumsLibElement) Marshal() string {
	return ""
}

func (e *AlbumsLibElement) Unmarshal(string) error {
	return nil
}

func (e *AlbumsLibElement) ChildAttributeID() int {
	return config.MTAttrAlbum
}

func (e *AlbumsLibElement) NewChild(value string) LibraryPathElement {
	c := NewAlbumLibElement()
	c.(*AlbumLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// AlbumLibElement
//----------------------------------------------------------------------------------------------------------------------

type AlbumLibElement struct {
	BaseAttrHolder
}

func NewAlbumLibElement() LibraryPathElement {
	return &AlbumLibElement{BaseAttrHolder{attrID: config.MTAttrAlbum}}
}

func (e *AlbumLibElement) Icon() string {
	return "ymuse-album"
}

func (e *AlbumLibElement) Label() string {
	if e.attrValue == "" {
		return glib.Local("(unknown)")
	}
	return e.attrValue
}

func (e *AlbumLibElement) IsFolder() bool {
	return true
}

func (e *AlbumLibElement) IsPlayable() bool {
	return true
}

func (e *AlbumLibElement) Prefix() string {
	return "album"
}

func (e *AlbumLibElement) Marshal() string {
	return e.attrValue
}

func (e *AlbumLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal AlbumLibElement: want 1 field, got %d", len(fields))
	}
	e.attrValue = fields[0]
	return nil
}

func (e *AlbumLibElement) ChildAttributeID() int {
	return config.MTAttrTrack
}

func (e *AlbumLibElement) NewChild(value string) LibraryPathElement {
	c := NewTrackLibElement()
	c.(*TrackLibElement).attrValue = value
	return c
}

//----------------------------------------------------------------------------------------------------------------------
// TrackLibElement
//----------------------------------------------------------------------------------------------------------------------

type TrackLibElement struct {
	BaseAttrHolder
}

func NewTrackLibElement() LibraryPathElement {
	return &TrackLibElement{BaseAttrHolder{attrID: config.MTAttrTrack}}
}

func (e *TrackLibElement) Icon() string {
	return "ymuse-audio-file"
}

func (e *TrackLibElement) Label() string {
	if e.attrValue == "" {
		return glib.Local("(unknown)")
	}
	return e.attrValue
}

func (e *TrackLibElement) IsFolder() bool {
	return false
}

func (e *TrackLibElement) IsPlayable() bool {
	return true
}

func (e *TrackLibElement) Prefix() string {
	return "track"
}

func (e *TrackLibElement) Marshal() string {
	return e.attrValue
}

func (e *TrackLibElement) Unmarshal(data string) error {
	fields := strings.Split(data, pathFieldSeparator)
	if len(fields) != 1 {
		return fmt.Errorf("failed to unmarshal TrackLibElement: want 1 field, got %d", len(fields))
	}
	e.attrValue = fields[0]
	return nil
}
