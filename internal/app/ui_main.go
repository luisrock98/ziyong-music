package app

import (
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

const navHeight = 64

func (a *app) view(c *ui.Context) {
	t := *c.Theme()
	a.pal = paletteFor(&t)
	p := a.pal
	t.Accent, t.AccentText = p.accent, p.onAccent
	t.Selection = p.accent.Alpha(0.3)
	c.SetTheme(&t)

	a.shortcuts(c)

	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	root := ui.Box(c).Fill()
	if !c.Focused(a.searchBox) {
		// Space plays and pauses wherever the focus is, as in Music. On
		// the container around everything, it comes before a button the
		// pointer focused, which would take Space as a click; on the
		// window, it would come after.
		root.OnShortcut(0, ui.KeySpace, a.pl.TogglePlay)
		// And where nothing has the focus, as once Now Playing covers the
		// button that had it.
		c.OnShortcut(0, ui.KeySpace, a.pl.TogglePlay)
	}
	npOpen, npSettled := a.npState(c)
	root.Children(func() {
		if !npOpen || !npSettled {
			ui.Box(c.Key("main")).Fill().Background(p.bg).Children(func() {
				a.content(c)
				a.pageTop(c)
				a.playerBar(c)
			})
		}
		a.nowPlayingLayer(c, npOpen, npSettled)
	})
	a.loginDialog(c)
	a.playlistNameDialog(c)
	a.deletePlaylistDialog(c)
	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.syncWindows()
}

func (a *app) shortcuts(c *ui.Context) {
	// The side buttons of a mouse, the keys of keyboards that have them
	// and swipes between pages, and the keys of Music.
	c.OnShortcut(0, ui.KeyBack, a.back)
	c.OnShortcut(0, ui.KeyForward, a.goForward)
	c.OnShortcut(ui.Cmd, ui.KeyBracketLeft, a.back)
	c.OnShortcut(ui.Cmd, ui.KeyBracketRight, a.goForward)
	c.OnShortcut(ui.Cmd, ui.KeyF, func() {
		a.nowPlaying = false
		a.searchBox.Focus(c)
	})
	if a.nowPlaying {
		c.OnShortcut(0, ui.KeyEscape, func() { a.nowPlaying = false })
		// ⌘ and a number choose Now Playing's tab of that place.
		if a.pl.Current() != nil {
			for i, t := range a.npTabs() {
				c.OnShortcut(ui.Cmd, ui.Key1+ui.Key(i), func() { a.choosePane(t.pane) })
			}
		}
	}
}

// go opens a page, remembering the one before for the back button, and
// closes Now Playing over it, though it is the page open.
func (a *app) open(r route) {
	a.nowPlaying = false
	if r == a.route {
		return
	}
	if r.page == pageArtist && a.lastArtist != 0 && a.lastArtist != r.id {
		a.artistTab = artistOverview
	}
	a.history, a.forward = append(a.history, a.route), nil
	a.show(r)
}

// show shows the page of r.
func (a *app) show(r route) {
	if a.route != r {
		a.descriptionOpen = false
		a.categoryPanel, a.editingCategories = false, false

	}
	if r.page == pageArtist {
		a.lastArtist = r.id
	}
	a.route = r
	if r.page == pageSearch {
		a.query, a.searched = r.query, r
	}
	a.load(r)
}

// load loads what the page of r shows, unless it has.
func (a *app) load(r route) {
	switch r.page {
	case pageLibrary:
		a.loadLibraryCollection()
		if id := a.likedPlaylist(); id != 0 {
			a.loadPlaylist(id, false)
		}
	case pageDiscover:
		a.loadDiscovery(r.query)
	case pageHome:
		a.freshenHome()
	case pagePlaylist:
		a.loadPlaylist(r.id, false)
	case pageSearch:
		a.loadSearch(r, false)
	case pageArtist:
		a.loadArtist(r.id)
	case pageAlbum:
		a.loadAlbum(r.id)
	case pageUser:
		a.loadUser(r.id)
	case pagePodcast:
		a.loadPodcast(r.id)
	case pagePodcasts:
		a.loadPodcastTab()
	}
}

// back goes to the page before, as the back button of a mouse does, or
// closes Now Playing, which covers the pages.
func (a *app) back() {
	if a.nowPlaying {
		a.nowPlaying = false
		return
	}
	if len(a.history) == 0 {
		return
	}
	a.forward = append(a.forward, a.route)
	r := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.show(r)
}

// goForward goes to the page back left, as the forward button of a mouse
// does.
func (a *app) goForward() {
	if len(a.forward) == 0 {
		return
	}
	a.history = append(a.history, a.route)
	r := a.forward[len(a.forward)-1]
	a.forward = a.forward[:len(a.forward)-1]
	a.nowPlaying = false
	a.show(r)
}

// playerBarHeight is the height of the player along the foot of the pages.
const playerBarHeight = 64

// pageGap is the room between the bar over a page and its content.
const pageGap = 32

// insets returns the room the bars over a page take at its top and its
// bottom: pages pad their content by it, and scroll under the bars.
func (a *app) insets(c *ui.Context) (top, bottom float32) {
	return max(navHeight, topBar(c)), playerBarHeight
}

// topEdgePad is the room the edge over a page leaves below the strip, on
// macOS.
const topEdgePad = 6

// buttonsHeight is the height of the window buttons off macOS, which fill
// the edge over a page.
const buttonsHeight = topBarHeight + topEdgePad

// topBar returns the height of the strip along the top of the window, as
// tall as the window buttons and the room above and below them, which
// what sits in it centers on.
func topBar(c *ui.Context) float32 {
	if runtime.GOOS != "darwin" {
		// The whole edge, which the buttons fill: what sits in the strip
		// centers on their glyphs.
		return buttonsHeight
	}
	return max(c.TitleBar().Height, topBarHeight)
}

// topBarHeight is the least height of the strip along the top.
const topBarHeight = 40

// startMark marks where a page's content starts, first in what scrolls:
// once it has moved up, content is under the bar at the top.
func (a *app) startMark(c *ui.Context) {
	b := ui.Box(c.Key("start-mark")).Height(0).Bounds()
	if b.W > 0 {
		top, _ := a.insets(c)
		a.underTop = b.Y < top+pageGap-0.5
	}
}

// endMark marks where a page's content ends, last in what scrolls: while it
// is below the top of the player, content is under the player.
func (a *app) endMark(c *ui.Context) {
	b := ui.Box(c.Key("end-mark")).Height(0).Bounds()
	if b.W > 0 {
		_, h := c.Size()
		_, bottom := a.insets(c)
		a.underBottom = b.Y > h-bottom+0.5
	}
}

// edgeShown returns how much of a scroll edge shows over el: fully while
// content is under its bar, and none at rest, as AppKit shows its edges.
func edgeShown(el ui.Element, under bool) float32 {
	target := float32(0)
	if under {
		target = 1
	}
	return el.AnimateTo("shown", target, ui.Animation{Duration: 180 * time.Millisecond})
}

// pageTop keeps navigation, search and the account within reach on every page.
func (a *app) pageTop(c *ui.Context) {
	top, _ := a.insets(c)
	p := a.pal
	edge := ui.Box(c.Key("top-edge")).Absolute().Top(0).Left(0).Right(0).Height(top).
		Material(glass.ScrollEdge{Hard: true, Background: p.bg})
	edge.Opacity(edgeShown(edge, a.underTop))
	bar := c.TitleBar()
	ui.Row(c.Key("navigation")).Absolute().Top(0).Left(0).Right(0).Height(top).
		Padding(0, max(pagePad, bar.Right+8), 0, max(pagePad, bar.Left+12)).Gap(14).
		AlignItems(ui.Center).DragWindow().Background(p.bg.Alpha(0.96)).Children(func() {
		ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Center).Children(func() {
			iconButton(c.Key("back"), iconBack, 17, p.text, "返回").Disabled(len(a.history) == 0).OnClick(a.back)
			iconButton(c.Key("forward"), iconForward, 17, p.text, "前进").Disabled(len(a.forward) == 0).OnClick(a.goForward)
		})
		ui.Row(c).Gap(4).Children(func() {
			for _, tab := range []struct {
				title string
				page  pageKind
			}{
				{"首页", pageHome}, {"发现", pageDiscover}, {"音乐库", pageLibrary},
			} {
				b := ui.ButtonBase(c.Key(tab.page)).Padding(7, 14).Center().Radius(9).Label(tab.title)
				ink := p.text
				if a.route.page == tab.page || tab.page == pageDiscover && (a.route.page == pageCharts || a.route.page == pagePodcasts) {
					ink = p.accent
				} else if b.Hovered() {
					b.Background(p.hover)
				}
				b.Children(func() { ui.Text(c, tab.title).FontSize(18).FontWeight(700).TextColor(ink) })
				b.OnClick(func() { a.open(route{page: tab.page}) })
			}
		})
		ui.Row(c).Grow(1).Basis(0).MinWidth(170).Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Width(190).MinWidth(100).Shrink(1).Children(func() { a.searchField(c) })
			a.accountRow(c)
		})
	})
}

func (a *app) content(c *ui.Context) {
	// A new page starts at its top. Not as it is opened: a control in a
	// list, as a page's tabs in its header, opens it as the list builds.
	if a.scrolled != a.route {
		a.trackList, a.pageScroll, a.scrolled = ui.ListState{}, ui.ScrollState{}, a.route
	}
	// Pages mark where their content is, as they scroll.
	a.underTop, a.underBottom = false, false
	switch a.route.page {
	case pageDiscover:
		a.discoverPage(c)
	case pageQueue:
		a.queuePage(c)
	case pageLibrary:
		a.libraryPage(c)
	case pageHome:
		a.homePage(c)
	case pageDaily:
		a.dailyPage(c)
	case pageCharts:
		a.chartsPage(c)
	case pagePlaylist:
		a.playlistPage(c)
	case pageSearch:
		a.searchPage(c)
	case pageFM:
		a.fmPage(c)
	case pageArtist:
		a.artistPage(c)
	case pageAlbum:
		a.albumPage(c)
	case pageUser:
		a.userPage(c)
	case pagePodcast:
		a.podcastPage(c)
	case pagePodcasts:
		a.podcastsPage(c)
	}
}

// centered shows a message, with a retry button if retry is set: what a
// page shows in place of what failed to load.
func (a *app) centered(c *ui.Context, err string, retry func()) {
	top, bottom := a.insets(c)
	ui.Column(c).Grow(1).Padding(top, 0, bottom, 0).Center().Gap(12).Children(func() {
		ui.Text(c, err).TextColor(a.pal.muted).FontSize(13)
		if retry != nil && ui.Button(c, "重试").Clicked() {
			retry()
		}
	})
}

// Loading what the pages show.

func (a *app) loadHome() {
	a.homeLoading, a.homeErr = true, ""
	a.homeGeneration++
	generation := a.homeGeneration
	go func() {
		albums, _ := netease.NewAlbums()
		a.post(func() { a.homeAlbums = albums })
	}()
	signedIn := a.profile != nil
	at := time.Now()
	go func() {
		featured, err1 := netease.FeaturedPlaylists(30)
		charts, err2 := netease.Charts()
		daily, _ := netease.DailySongs()
		var rec []netease.Playlist
		var artists []netease.ArtistProfile
		if signedIn {
			artists = dailyArtists(daily)
			rec, _ = netease.RecommendedPlaylists()
		}
		a.post(func() {
			if generation != a.homeGeneration {
				return
			}
			a.homeLoading = false
			a.daily, a.homeArtists = daily, artists
			a.loadHomeArtistPhotos(artists, generation)
			if err1 != nil && err2 != nil {
				// Loading the new day's again, what shows stays, to try
				// once more.
				if len(a.featured) == 0 && len(a.charts) == 0 {
					a.homeErr = "无法连接网易云音乐，请检查网络"
				}
				return
			}
			a.featured, a.charts, a.recPlaylists = featured, charts, rec
			a.homeAt = at
		})
	}()
}

func (a *app) loadHomeArtistPhotos(artists []netease.ArtistProfile, generation int) {
	for i, artist := range artists {
		go func() {
			profile, err := netease.ArtistSummary(artist.ID)
			if err != nil || profile.ID != artist.ID || (profile.Avatar == "" && profile.PicURL == "") {
				return
			}
			a.post(func() {
				if generation == a.homeGeneration {
					a.homeArtists[i].Avatar = profile.Avatar
					a.homeArtists[i].PicURL = profile.PicURL
				}
			})
		}()
	}
}

// dailyArtists keeps the first six distinct, navigable artist credits in recommendation order.
func dailyArtists(tracks []netease.Track) []netease.ArtistProfile {
	seen := make(map[int64]bool)
	var artists []netease.ArtistProfile
	for _, track := range tracks {
		for _, artist := range track.Artists {
			if artist.ID <= 0 || strings.TrimSpace(artist.Name) == "" || seen[artist.ID] {
				continue
			}
			seen[artist.ID] = true
			artists = append(artists, netease.ArtistProfile{Artist: artist, PicURL: track.Album.PicURL})
			if len(artists) == 6 {
				return artists
			}
		}
	}
	return artists
}

// freshenHome loads the home page again once NetEase has picked the day's
// songs anew since it last did: the app may stay open overnight.
func (a *app) freshenHome() {
	if !a.homeLoading && !a.homeAt.IsZero() && dailyRenewed(a.homeAt, time.Now()) {
		a.loadHome()
	}
}

// beijing is NetEase's time, without daylight saving.
var beijing = time.FixedZone("CST", 8*60*60)

// dailyRenewed reports whether NetEase has picked new daily songs between
// loaded and now. It does at 6:00, Beijing time.
func dailyRenewed(loaded, now time.Time) bool {
	b := now.In(beijing)
	six := time.Date(b.Year(), b.Month(), b.Day(), 6, 0, 0, 0, beijing)
	if b.Before(six) {
		six = six.AddDate(0, 0, -1)
	}
	return loaded.Before(six)
}

func (a *app) loadPlaylist(id int64, thenPlay bool) {
	if pl := a.playlists[id]; pl != nil {
		if thenPlay {
			a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
		}
		return
	}
	if a.loadingPl[id] {
		return
	}
	a.loadingPl[id] = true
	delete(a.plErr, id)
	go func() {
		pl, err := netease.PlaylistDetail(id)
		a.post(func() {
			delete(a.loadingPl, id)
			if err != nil {
				a.plErr[id] = "歌单加载失败"
				return
			}
			a.playlists[id] = pl
			if thenPlay {
				a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
			}
		})
	}()
}

func firstPlayable(tracks []netease.Track) int {
	for i := range tracks {
		if tracks[i].Playable() {
			return i
		}
	}
	return 0
}
