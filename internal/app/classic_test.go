package app

import (
	"github.com/egoist/mygo/ui"
	"netease-music-mygo/internal/netease"
	"testing"
)

// Opening a queue and its current song preserves the queue; choosing an upcoming song jumps within it.
func TestClassicQueueNavigation(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]netease.Track{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}, {ID: 3, Name: "Third"}})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click("播放队列"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route.page != pageQueue {
		t.Fatal("queue button did not open its page")
	}
	if err := tt.Click("First"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.nowPlaying || len(a.pl.Queue()) != 3 {
		t.Fatal("current song changed the queue")
	}
	a.nowPlaying = false
	tt.Frame()
	tt.Advance(npSettle)
	tt.Frame()
	if err := tt.Click("Third"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pl.Current().ID != 3 || len(a.pl.Queue()) != 3 {
		t.Fatal("queue selection did not jump in place")
	}
}

// Search overview cards open the matching artist, and Back restores the same query.
func TestSearchOverviewNavigation(t *testing.T) {
	a := catalogApp()
	r := route{page: pageSearch, query: "MyGO", kind: searchOverview}
	artists := a.searches[route{page: pageSearch, query: "MyGO", kind: netease.SearchArtists}].res.Artists
	a.searches[r] = &searchData{res: netease.SearchResult{Artists: artists}, page: paged{loaded: true}}
	a.open(r)
	tt := ui.NewTester(a.view, 1200, 900)
	tt.Frame()
	if err := tt.Click("MyGO!!!!!"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route != (route{page: pageArtist, id: 7}) {
		t.Fatalf("opened %+v", a.route)
	}
	tt.Click("返回")
	tt.Frame()
	if a.route != r {
		t.Fatalf("back lost the overview query: %+v", a.route)
	}
}

// Daily picks include collaborators, keep distinct namesakes, and count only valid unique IDs toward six.
func TestDailyArtists(t *testing.T) {
	tracks := []netease.Track{
		{Artists: []netease.Artist{{ID: 0, Name: "Unknown"}, {ID: 1, Name: "A"}, {ID: 2, Name: "B"}}},
		{Artists: []netease.Artist{{ID: 1, Name: "Alias"}, {ID: 3, Name: "B"}, {ID: 4, Name: " "}}},
		{Artists: []netease.Artist{{ID: 5, Name: "C"}, {ID: 6, Name: "D"}, {ID: 7, Name: "E"}, {ID: 8, Name: "F"}}},
	}
	got := dailyArtists(tracks)
	want := []int64{1, 2, 3, 5, 6, 7}
	if len(got) != len(want) {
		t.Fatalf("picked %d artists, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("artist at %d = %d, want %d", i, got[i].ID, id)
		}
	}
}

// Pinning a genre keeps it on the discovery bar after closing the editor, and selecting it opens its catalog.
func TestPinDiscoveryCategory(t *testing.T) {
	a := newTestApp()
	a.discovery = map[string]*discoveryData{"全部": {page: paged{loaded: true}}, "爵士": {page: paged{loaded: true}}}
	a.open(route{page: pageDiscover})
	tt := ui.NewTester(a.view, 1200, 900)
	for _, label := range []string{"更多分类", "编辑常用分类", "爵士", "完成编辑", "更多分类", "爵士"} {
		tt.Frame()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	tt.Frame()
	if a.route.query != "爵士" || a.categoryPanel {
		t.Fatalf("selected route %+v, panel open %v", a.route, a.categoryPanel)
	}
}

// A saved album opens its detail, where clicking the cover starts the album's queue.
func TestLibraryAlbumCoverPlayback(t *testing.T) {
	a := catalogApp()
	a.profile = &netease.Profile{UserID: 9}
	a.libraryCollection = &libraryCollection{albums: []netease.AlbumProfile{*a.albums[8].album}, albumPage: paged{loaded: true}}
	a.open(route{page: pageLibrary})
	tt := ui.NewTester(a.view, 1200, 900)
	for _, label := range []string{"收藏的专辑", "迷跡波", "播放封面中的曲目"} {
		tt.Frame()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	tt.Frame()
	songs := a.albums[8].album.Songs
	if a.route != (route{page: pageAlbum, id: 8}) || a.pl.Current() == nil || a.pl.Current().ID != songs[0].ID || len(a.pl.Queue()) != len(songs) {
		t.Fatal("album cover did not play the selected album")
	}
}

// Switching from the wall to the stage and then a comment author's page preserves the current song.
func TestImmersiveCommentAuthor(t *testing.T) {
	a := catalogApp()
	tr := a.albums[8].album.Songs[0]
	a.pl.Cue([]netease.Track{tr})
	a.pl.SetLyrics(nil)
	a.nowPlaying = true
	cm := netease.Comment{ID: 33, Content: "喜欢这首歌"}
	cm.User.ID, cm.User.Nickname = 9, "听歌的人"
	a.comments = commentState{track: tr.ID, loaded: true, total: 1, latest: []netease.Comment{cm}}
	tt := ui.NewTester(a.view, 880, 580)
	for _, name := range []string{"照片墙", "舞台", "评论 1", "听歌的人"} {
		tt.Frame()
		tt.Advance(npSettle)
		tt.Frame()
		if err := tt.Click(name); err != nil {
			t.Fatal(err)
		}
	}
	tt.Frame()
	if a.route != (route{page: pageUser, id: 9}) || a.nowPlaying || a.pl.Current().ID != tr.ID {
		t.Fatalf("opening author left route %+v, now playing %v", a.route, a.nowPlaying)
	}
}

// Filtering to subscribed podcasts still opens the selected show without starting its episodes.
func TestPodcastSubscriptionFilter(t *testing.T) {
	a := catalogApp()
	a.open(route{page: pagePodcasts})
	tt := ui.NewTester(a.view, 1000, 800)
	tt.Frame()
	if err := tt.Click("我的订阅"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.podcastTab.filter != "我的订阅" {
		t.Fatal("subscription filter was not applied")
	}
	if err := tt.Click("夜航电台"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.route != (route{page: pagePodcast, id: 5}) || a.pl.Current() != nil {
		t.Fatalf("podcast card opened %+v", a.route)
	}
}
