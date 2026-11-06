package netease

import (
	"net/url"
	"strconv"
)

// CategoryPlaylists returns a page of playlists in a NetEase category.
func CategoryPlaylists(category string, limit, offset int) ([]Playlist, bool, error) {
	var res struct {
		Playlists []Playlist `json:"playlists"`
		More      bool       `json:"more"`
	}
	q := url.Values{"cat": {category}, "order": {"hot"}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}, "total": {"true"}}
	err := get("/api/playlist/list", q, &res)
	return res.Playlists, res.More, err
}

// ArtistSummary returns card artwork without loading song details or privileges.
func ArtistSummary(id int64) (ArtistProfile, error) {
	var res struct {
		Artist ArtistProfile `json:"artist"`
	}
	err := get("/api/v1/artist/"+strconv.FormatInt(id, 10), nil, &res)
	return res.Artist, err
}

// NewAlbums returns the newest albums for the home page.
func NewAlbums() ([]AlbumProfile, error) {
	var res struct {
		Albums []AlbumProfile `json:"albums"`
	}
	err := get("/api/album/new", url.Values{"area": {"ALL"}, "limit": {"10"}, "offset": {"0"}, "total": {"true"}}, &res)
	return res.Albums, err
}
