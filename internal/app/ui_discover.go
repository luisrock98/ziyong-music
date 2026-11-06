package app

import (
	"github.com/egoist/mygo/ui"
	"netease-music-mygo/internal/netease"
	"slices"
)

type discoveryData struct {
	lists []netease.Playlist
	page  paged
}

var categoryPlaylists = netease.CategoryPlaylists

func (a *app) loadDiscovery(category string) {
	if category == "推荐歌单" || category == "排行榜" {
		if len(a.featured) == 0 && !a.homeLoading {
			a.loadHome()
		}
		return
	}
	if category == "" {
		category = "全部"
	}
	if a.discovery == nil {
		a.discovery = map[string]*discoveryData{}
	}
	d := a.discovery[category]
	if d == nil {
		d = &discoveryData{}
		a.discovery[category] = d
	}
	if !d.page.loaded {
		a.moreDiscovery(category, d)
	}
}

func (a *app) moreDiscovery(category string, d *discoveryData) {
	if d.page.loading {
		return
	}
	d.page.loading, d.page.err = true, ""
	offset := len(d.lists)
	go func() {
		lists, more, err := categoryPlaylists(category, 30, offset)
		a.post(func() {
			d.page.loading = false
			if err != nil {
				d.page.err = "歌单加载失败，请重试"
				return
			}
			d.lists = append(d.lists, lists...)
			d.page.loaded, d.page.more = true, more && len(lists) > 0
		})
	}()
}

func (a *app) playlistGrid(c *ui.Context, lists []netease.Playlist) {
	ui.Grid(c).Columns(a.columns(c, 160)).GapX(24).GapY(40).Children(func() {
		for i := range lists {
			a.playlistCard(c.Key(lists[i].ID), &lists[i])
		}
	})
}

func (a *app) discoverPage(c *ui.Context) {
	a.scrollPage(c, func() {
		a.heading(c, "发现", "")
		category := a.route.query
		if category == "" {
			category = "全部"
		}
		ui.Row(c).Wrap().Gap(8).Margin(12, 0, 28, 0).Children(func() {
			cats := append([]string{"全部", "推荐歌单", "排行榜"}, a.pinnedCategories()...)
			if !slices.Contains(cats, category) {
				cats = append(cats, category)
			}
			for _, cat := range cats {
				filterButton(c.Key(cat), cat, category == cat).OnClick(func() { a.open(route{page: pageDiscover, query: cat}) })
			}
			filterButton(c.Key("more-categories"), "更多分类", a.categoryPanel).OnClick(func() { a.categoryPanel = !a.categoryPanel; a.editingCategories = false })
			filterButton(c.Key("podcasts"), "播客", false).OnClick(func() { a.open(route{page: pagePodcasts}) })
		})
		if a.categoryPanel {
			a.discoveryCategories(c)
		}
		switch category {
		case "排行榜":
			a.chartSections(c)
		case "推荐歌单":
			lists := a.recPlaylists
			if len(lists) == 0 {
				lists = a.featured
			}
			a.playlistGrid(c, lists)
		default:
			d := a.discovery[category]
			if d == nil {
				return
			}
			if !d.page.loaded && d.page.loading {
				a.playlistCardsSkeleton(c, a.pageShimmer(c), 10, 160, false)
				return
			}
			a.playlistGrid(c, d.lists)
			a.pageFoot(c, &d.page, func() { a.moreDiscovery(category, d) })
			if d.page.loaded && len(d.lists) == 0 {
				a.empty(c, "暂无歌单")
			}
		}
		if a.homeLoading && (category == "推荐歌单" || category == "排行榜") {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), 10, 160, false)
		}
		if a.homeErr != "" && (category == "推荐歌单" || category == "排行榜") {
			ui.Button(c, "重新加载").OnClick(a.loadHome)
		}
	})
}

// compactTracks provides the same queue behavior in library and search previews.
func (a *app) compactTracks(c *ui.Context, tracks []netease.Track, columns int) {
	ui.Grid(c).Columns(columns).GapX(14).GapY(4).Children(func() {
		for i := range tracks {
			tr := &tracks[i]
			b := ui.ButtonBase(c.Key(tr.ID)).Height(56).Padding(6).Gap(10).Radius(8).AlignItems(ui.Center).Label(tr.Name).ContextMenu(a.songItems(tr, 0))
			if cur := a.pl.Current(); cur != nil && cur.ID == tr.ID {
				b.Background(a.pal.primaryBG()).TextColor(a.pal.accent)
			} else if b.Hovered() {
				b.Background(a.pal.hover)
			}
			if !tr.Playable() {
				b.Opacity(.4)
			}
			b.Children(func() {
				a.cover(c, tr.Album.PicURL, 100, 40, 6)
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, tr.Name).FontSize(14).FontWeight(650).SingleLine()
					ui.Row(c).Children(func() {
						a.artistLinks(c, tr, a.pal.muted, a.pal.text, a.route.page == pageArtist).FontSize(12).SingleLine().Shrink(1).MinWidth(0)
					})
				})
			})
			b.OnClick(func() {
				if tr.Playable() {
					a.pl.PlayQueue(tracks, i)
				}
			})
		}
	})
}

func (a *app) queuePage(c *ui.Context) {
	// trackRows is shared for its menus and seeking to a chosen song, while
	// queuePane retains jump-in-place controls in the immersive view.
	a.scrollPage(c, func() {
		a.heading(c, "正在播放", "")
		if tr := a.pl.Current(); tr != nil {
			ui.ButtonBase(c.Key("current")).Height(64).Gap(14).AlignItems(ui.Center).Label(tr.Name).Children(func() {
				a.cover(c, tr.Album.PicURL, 100, 46, 8)
				ui.Text(c, tr.Name).FontSize(18).FontWeight(650)
			}).OnClick(func() { a.nowPlaying = true })
		}
		a.section(c, "接下来播放")
		order, at := a.pl.PlayOrder()
		for _, i := range order[max(0, at+1):] {
			tr := a.pl.Queue()[i]
			b := ui.ButtonBase(c.Key(i)).Height(62).Gap(14).Padding(8).Radius(8).AlignItems(ui.Center).Label(tr.Name).ContextMenu(a.songItems(&tr, 0))
			if b.Hovered() {
				b.Background(a.pal.hover)
			}
			b.Children(func() {
				a.cover(c, tr.Album.PicURL, 100, 46, 8)
				ui.Text(c, tr.Name).Grow(1).FontSize(16).SingleLine()
				ui.Text(c, tr.ArtistNames()).TextColor(a.pal.muted).SingleLine()
				ui.Text(c, clock(tr.Length())).TextColor(a.pal.muted)
			})
			b.OnClick(func() {
				if tr.Playable() {
					a.pl.Jump(i)
				}
			})
		}
	})
}

func (a *app) pinnedCategories() []string {
	if a.settings.PinnedCategories != nil {
		return a.settings.PinnedCategories
	}
	return []string{"华语", "欧美", "流行", "摇滚", "电子", "轻音乐"}
}

func (a *app) discoveryCategories(c *ui.Context) {
	groups := []struct {
		title string
		tags  []string
	}{
		{"语种", []string{"华语", "欧美", "日语", "韩语", "粤语"}},
		{"风格", []string{"流行", "摇滚", "电子", "轻音乐", "民谣", "爵士", "古典", "说唱", "独立", "R&B/Soul", "金属", "乡村"}},
		{"场景", []string{"清晨", "夜晚", "学习", "工作", "运动", "旅行", "驾车"}},
		{"情感", []string{"怀旧", "清新", "浪漫", "伤感", "治愈", "放松", "快乐", "孤独"}},
		{"主题", []string{"影视原声", "ACG", "游戏", "经典", "翻唱", "现场", "器乐"}},
	}
	ui.Column(c.Key("category-panel")).Background(a.pal.hover).Radius(16).Padding(20).Gap(12).Margin(0, 0, 28, 0).Children(func() {
		label := "编辑常用分类"
		if a.editingCategories {
			label = "完成编辑"
		}
		filterButton(c, label, a.editingCategories).AlignSelf(ui.End).OnClick(func() { a.editingCategories = !a.editingCategories })
		for _, g := range groups {
			ui.Row(c.Key(g.title)).Gap(14).AlignItems(ui.Start).Children(func() {
				ui.Text(c, g.title).FontWeight(700).Padding(8, 0)
				ui.Row(c).Grow(1).Basis(0).Wrap().Gap(6).Children(func() {
					for _, tag := range g.tags {
						active := tag == a.route.query
						if a.editingCategories {
							active = slices.Contains(a.pinnedCategories(), tag)
						}
						filterButton(c.Key(tag), tag, active).OnClick(func() {
							if !a.editingCategories {
								a.open(route{page: pageDiscover, query: tag})
								a.categoryPanel = false
								return
							}
							pinned := append([]string{}, a.pinnedCategories()...)
							if i := slices.Index(pinned, tag); i >= 0 {
								pinned = slices.Delete(pinned, i, i+1)
							} else {
								pinned = append(pinned, tag)
							}
							a.settings.PinnedCategories = pinned
							a.settings.save()
						})
					}
				})
			})
		}
	})
}
