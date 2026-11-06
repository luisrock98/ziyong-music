package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// searchResults and searchSuggest search NetEase; tests replace them.
var (
	searchResults        = netease.SearchPage
	searchSuggest        = netease.SearchSuggest
	searchPreviewResults = netease.SearchPage
)

// searchLimit is how many results load at a time.
const searchLimit = 60
const searchOverview netease.SearchType = -1

// searchKinds are the kinds of search, as their tabs show them.
var searchKinds = []struct {
	kind        netease.SearchType
	label, unit string
}{
	{searchOverview, "综合", "项结果"},
	{netease.SearchSongs, "单曲", "首单曲"},
	{netease.SearchArtists, "歌手", "位歌手"},
	{netease.SearchAlbums, "专辑", "张专辑"},
	{netease.SearchPlaylists, "歌单", "个歌单"},
	{netease.SearchUsers, "用户", "位用户"},
	{netease.SearchPodcasts, "播客", "个播客"},
}

// searchData is what a search found, as far as its pages have loaded.
type searchData struct {
	res             netease.SearchResult
	page            paged
	pendingPreviews int
	previewError    bool
}

// search searches for q, of the kind last searched for.
func (a *app) search(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	kind := a.searched.kind
	if kind == 0 {
		kind = searchOverview
	}
	a.searchFor(route{page: pageSearch, query: q, kind: kind})
}

// searchFor opens the page of a search, remembering its query. Searched
// for again, it loads anew.
func (a *app) searchFor(r route) {
	a.rememberSearch(r.query)
	a.dismissSuggestions()
	if r == a.route {
		a.nowPlaying = false
		a.loadSearch(r, true)
		return
	}
	a.open(r)
}

// loadSearch loads the first page of a search, unless it has, or again.
func (a *app) loadSearch(r route, again bool) {
	if a.searches[r] != nil && !again {
		return
	}
	d := &searchData{}
	a.searches[r] = d
	a.moreResults(r, d)
	if r.kind == searchOverview {
		fetch := searchPreviewResults
		for _, kind := range []netease.SearchType{netease.SearchArtists, netease.SearchAlbums, netease.SearchPlaylists} {
			d.pendingPreviews++
			go func() {
				limit := 3
				if kind == netease.SearchPlaylists {
					limit = 5
				}
				res, err := fetch(r.query, kind, limit, 0)
				a.post(func() {
					d.pendingPreviews--
					if err != nil {
						d.previewError = true
						return
					}
					switch kind {
					case netease.SearchArtists:
						d.res.Artists = res.Artists
					case netease.SearchAlbums:
						d.res.Albums = res.Albums
					case netease.SearchPlaylists:
						d.res.Playlists = res.Playlists
					}
				})
			}()
		}
	}
}

// moreResults loads the next page of a search. A search done again has
// new data, so the last one's pages land where nothing shows them.
func (a *app) moreResults(r route, d *searchData) {
	if d.page.loading {
		return
	}
	d.page.loading, d.page.err = true, ""
	offset := d.res.Len()
	go func() {
		kind, limit := r.kind, searchLimit
		if kind == searchOverview {
			kind, limit = netease.SearchSongs, 12
		}
		res, err := searchResults(r.query, kind, limit, offset)
		a.post(func() {
			d.page.loading = false
			if err != nil {
				d.page.err = "搜索失败，请稍后再试"
				return
			}
			d.res.Append(res)
			// A server can claim more while sending none.
			d.page.loaded, d.page.more = true, res.Len() > 0 && d.res.Len() < d.res.Total && r.kind != searchOverview
		})
	}()
}

// searchHistoryLimit is how many searches the history keeps.
const searchHistoryLimit = 20

// rememberSearch puts q first in the history of searches, once.
func (a *app) rememberSearch(q string) {
	history := []string{q}
	for _, s := range a.settings.SearchHistory {
		if s != q && len(history) < searchHistoryLimit {
			history = append(history, s)
		}
	}
	a.settings.SearchHistory = history
	a.settings.save()
}

func (a *app) clearSearchHistory() {
	a.settings.SearchHistory = nil
	a.settings.save()
}

func (a *app) searchPage(c *ui.Context) {
	r := a.route
	d := a.searches[r]
	if r.kind == searchOverview {
		a.searchOverviewPage(c, d)
		return
	}
	header := func(c *ui.Context) { a.searchHeader(c, d) }
	more := func() { a.moreResults(r, d) }
	if r.kind == netease.SearchSongs {
		if !d.page.loaded && d.page.loading {
			a.trackSkeleton(c, a.pageShimmer(c), true, header)
			return
		}
		a.trackRows(c, d.res.Songs, true, header, func(c *ui.Context) {
			a.pageFoot(c, &d.page, more)
			a.searchEmpty(c, d)
		})
		return
	}
	a.scrollPage(c, func() {
		header(c)
		res := &d.res
		switch r.kind {
		case netease.SearchArtists:
			w, _ := c.Size()
			cols := max(2, min(6, int((w-2*pagePad+24)/156)))
			ui.Grid(c).Columns(cols).GapX(24).GapY(32).Children(func() {
				for i := range res.Artists {
					a.artistCard(c.Key(res.Artists[i].ID), &res.Artists[i])
				}
			})
		case netease.SearchAlbums:
			a.albumGrid(c, res.Albums)
		case netease.SearchPlaylists:
			a.playlistGrid(c, res.Playlists)
		case netease.SearchUsers:
			ui.Grid(c).Columns(min(3, a.columns(c, 320))).Gap(12).Children(func() {
				for i := range res.Users {
					a.userRow(c.Key(res.Users[i].UserID), &res.Users[i])
				}
			})
		case netease.SearchPodcasts:
			ui.Grid(c).Columns(min(3, a.columns(c, 320))).Gap(12).Children(func() {
				for i := range res.Podcasts {
					a.podcastRow(c.Key(res.Podcasts[i].ID), &res.Podcasts[i])
				}
			})
		}
		if !d.page.loaded && d.page.loading {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 156), 156, false)
			return
		}
		a.pageFoot(c, &d.page, more)
		a.searchEmpty(c, d)
	})
}

func (a *app) searchEmpty(c *ui.Context, d *searchData) {
	if d.page.loaded && d.pendingPreviews == 0 && !d.previewError && d.res.Len() == 0 {
		a.empty(c, "没有找到相关结果")
	}
}

// searchHeader is what was searched for, how many it found, and the tabs
// of what it may look for instead.
func (a *app) searchHeader(c *ui.Context, d *searchData) {
	r := a.route
	sub := "搜索"
	for _, k := range searchKinds {
		if k.kind == r.kind {
			if d.page.loaded && r.kind != searchOverview {
				sub = fmt.Sprintf("%s搜索 · %d %s", k.label, d.res.Total, k.unit)
			}
		}
	}
	a.heading(c, "“"+r.query+"”", sub)

	ui.Row(c).Wrap().Gap(8).Margin(4, 0, 28, 0).Children(func() {
		for _, k := range searchKinds {
			filterButton(c.Key(k.kind), k.label, r.kind == k.kind).OnClick(func() { r.kind = k.kind; a.show(r) })
		}
	})

}

// artistCard is an artist's round photo over their name, opening their page.
func (a *app) artistCard(c *ui.Context, ar *netease.ArtistProfile) {
	p := a.pal
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(ar.Name)
	hovered := card.Hovered()
	card.Children(func() {
		photo := ar.Avatar
		if photo == "" {
			photo = ar.PicURL
		}
		art := ui.Box(c).AspectRatio(1).Radius(999)
		if hovered {
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.Children(func() { a.coverFill(c, photo, 400, 999) })
		ui.Text(c, ar.Name).FontSize(13).FontWeight(500).SingleLine().AlignSelf(ui.Center)
	})
	if card.Clicked() {
		a.open(route{page: pageArtist, id: ar.ID})
	}
}

func (a *app) searchOverviewPage(c *ui.Context, d *searchData) {
	a.scrollPage(c, func() {
		a.searchHeader(c, d)
		w, _ := c.Size()
		cols := 1
		if w >= 960 && len(d.res.Artists) > 0 && len(d.res.Albums) > 0 {
			cols = 2
		}
		ui.Grid(c).Columns(cols).Gap(36).Children(func() {
			if len(d.res.Artists) > 0 {
				ui.Column(c.Key("artists")).Children(func() {
					a.searchSection(c, "艺人", netease.SearchArtists)
					ui.Grid(c).Columns(3).Gap(20).MaxWidth(520).Children(func() {
						for i := range d.res.Artists {
							a.artistCard(c.Key(d.res.Artists[i].ID), &d.res.Artists[i])
						}
					})
				})
			}
			if len(d.res.Albums) > 0 {
				ui.Column(c.Key("albums")).Children(func() {
					a.searchSection(c, "专辑", netease.SearchAlbums)
					ui.Grid(c).Columns(3).Gap(20).MaxWidth(520).Children(func() {
						for i := range d.res.Albums {
							a.albumCard(c.Key(d.res.Albums[i].ID), &d.res.Albums[i])
						}
					})
				})
			}
		})
		if len(d.res.Songs) > 0 {
			ui.Column(c).Margin(36, 0, 0, 0).Children(func() {
				a.searchSection(c, "歌曲", netease.SearchSongs)
				a.compactTracks(c, d.res.Songs, min(3, a.columns(c, 240)))
			})
		}
		if len(d.res.Playlists) > 0 {
			ui.Column(c).Margin(36, 0, 0, 0).Children(func() { a.searchSection(c, "歌单", netease.SearchPlaylists); a.playlistGrid(c, d.res.Playlists) })
		}
		if d.page.loading || d.pendingPreviews > 0 {
			ui.Text(c, "正在搜索…").Margin(24, 0).TextColor(a.pal.muted)
		}
		if d.page.err != "" || d.previewError {
			ui.Button(c, "重新搜索").OnClick(func() { a.loadSearch(a.route, true) })
		}
		a.searchEmpty(c, d)
	})
}
func (a *app) searchSection(c *ui.Context, title string, kind netease.SearchType) {
	ui.Row(c).Margin(0, 0, 20, 0).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(24).FontWeight(700).Grow(1)
		filterButton(c, "查看全部"+title, false).OnClick(func() { a.open(route{page: pageSearch, query: a.route.query, kind: kind}) })
	})
}
