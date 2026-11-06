package app

import (
	"github.com/egoist/mygo/ui"
	"netease-music-mygo/internal/netease"
	"testing"
)

// Filtering the library changes its cards without touching the queue, and a card still opens its playlist.
func TestLibraryPlaylistFilters(t *testing.T) {
	a := editTestApp()
	a.playlists[11] = &netease.Playlist{ID: 11, Name: "睡前"}
	a.pl.Cue(a.playlists[10].Tracks)
	tt := ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	if err := tt.Click("音乐库"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("收藏的歌单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.libraryFilter != 2 || len(a.pl.Queue()) != 2 {
		t.Fatal("filter changed queue or was not selected")
	}
	if _, ok := tt.Find("睡前"); ok {
		t.Fatal("created playlist remained in saved filter")
	}
	if err := tt.Click("我的歌单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("睡前"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route != (route{page: pagePlaylist, id: 11}) {
		t.Fatalf("opened %+v", a.route)
	}
}

// Creating a playlist from the saved filter opens the dialog while preserving the filter.
func TestNewPlaylistFromLibraryFilter(t *testing.T) {
	a := editTestApp()
	tt := ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	tt.Click("音乐库")
	tt.Frame()
	tt.Click("收藏的歌单")
	tt.Frame()
	if err := tt.Click("新建歌单"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.naming.open || a.libraryFilter != 2 {
		t.Fatal("creating a playlist lost the selected filter")
	}
}
