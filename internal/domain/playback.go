package domain

// PlaybackState represents the current state of Spotify playback.
type PlaybackState string

const (
	PlaybackStatePlaying PlaybackState = "playing"
	PlaybackStatePaused  PlaybackState = "paused"
	PlaybackStateIdle    PlaybackState = "idle"
)

// MatchType represents the type of search result match.
type MatchType string

const (
	MatchTypeTrack  MatchType = "track"
	MatchTypeAlbum  MatchType = "album"
	MatchTypeArtist MatchType = "artist"
)

// Track represents a Spotify track.
type Track struct {
	URI    string
	Title  string
	Artist string
	Album  string
}

// Album represents a Spotify album.
type Album struct {
	URI    string
	Name   string
	Artist string
}

// Artist represents a Spotify artist.
type Artist struct {
	URI  string
	Name string
}

// MatchResult holds the top search result with its match type.
type MatchResult struct {
	Type   MatchType
	Track  *Track
	Album  *Album
	Artist *Artist
}

// PlaybackURI returns the Spotify URI for the matched result.
func (m *MatchResult) PlaybackURI() string {
	switch m.Type {
	case MatchTypeTrack:
		return m.Track.URI
	case MatchTypeAlbum:
		return m.Album.URI
	case MatchTypeArtist:
		return m.Artist.URI
	}
	return ""
}

// Device represents a Spotify playback device.
type Device struct {
	ID       string
	Name     string
	Type     string
	IsLocal  bool
	IsActive bool
}

// CurrentPlayback holds the current playback status.
type CurrentPlayback struct {
	State  PlaybackState
	Device *Device
	Track  *Track
}
