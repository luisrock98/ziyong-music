package netease

import (
	"os"
	"testing"
)

// The discovery endpoints return usable playlist, artist and album identities from the live service.
func TestDiscoveryLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip()
	}
	lists, _, err := CategoryPlaylists("欧美", 5, 0)
	if err != nil || len(lists) == 0 {
		t.Fatalf("playlists: %v, %d", err, len(lists))
	}
	artist, err := ArtistSummary(2116)
	if err != nil || artist.ID != 2116 || (artist.Avatar == "" && artist.PicURL == "") {
		t.Fatalf("artist summary: %v, artist %d", err, artist.ID)
	}
	albums, err := NewAlbums()
	if err != nil || len(albums) == 0 {
		t.Fatalf("albums: %v, %d", err, len(albums))
	}
	if lists[0].ID == 0 || albums[0].ID == 0 {
		t.Fatal("missing catalog identity")
	}
}
