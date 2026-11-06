package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/player"
)

const pagePad = 72

// columns returns how many cards of at least min DIPs fit across the page.
func (a *app) columns(c *ui.Context, minimum float32) int {
	w, _ := c.Size()
	inner := w - 2*pagePad
	return min(5, max(2, int((inner+24)/(minimum+24))))
}

func (a *app) heading(c *ui.Context, title, sub string) {
	ui.Column(c).Gap(4).Margin(0, 0, 18, 0).Children(func() {
		if sub != "" {
			ui.Text(c, sub).FontSize(12).FontWeight(600).TextColor(a.pal.muted).LetterSpacing(0.4)
		}
		ui.Text(c, title).FontSize(40).FontWeight(750).LetterSpacing(-0.8)
	})
}

func (a *app) section(c *ui.Context, title string) ui.Element {
	return ui.Text(c, title).FontSize(28).FontWeight(700).LetterSpacing(-0.5).Margin(46, 0, 20, 0)
}

func (a *app) homePage(c *ui.Context) {
	loading := len(a.featured) == 0 && len(a.charts) == 0
	now := c.Now()
	top, bottom := a.insets(c)
	ui.Scroll(c.Key("home")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		if loading && a.homeLoading {
			a.homeSkeleton(c, a.pageShimmer(c))
			return
		}
		picks := a.recPlaylists
		if len(picks) == 0 {
			picks = a.featured
		}
		if len(picks) > 0 {
			ui.Row(c).AlignItems(ui.Center).Margin(0, 0, 20, 0).Children(func() {
				ui.Text(c, "推荐歌单").FontSize(28).FontWeight(700).Grow(1)
				filterButton(c, "查看全部", false).OnClick(func() { a.open(route{page: pageDiscover, query: "推荐歌单"}) })
			})
			a.playlistGrid(c.Key("recommendations"), picks[:min(len(picks), a.columns(c, 160)*2)])
		}
		if len(picks) > 0 {
			a.section(c, "For You")
		}
		ui.Grid(c.Key("featured-stations")).Columns(2).Gap(24).Children(func() { a.dailyCard(c, now); a.fmHomeCard(c) })
		if loading {
			ui.Column(c).Padding(24).AlignItems(ui.Center).Gap(12).Children(func() {
				ui.Text(c, a.homeErr).TextColor(a.pal.muted)
				ui.Button(c, "重试").OnClick(a.loadHome)
			})
		}
		if len(a.homeArtists) > 0 {
			a.section(c, "为你推荐的艺人")
			w, _ := c.Size()
			cols := 6
			if w < 900 {
				cols = 3
			}
			ui.Grid(c).Columns(cols).Gap(24).Children(func() {
				for i := range a.homeArtists {
					a.artistCard(c.Key(a.homeArtists[i].ID), &a.homeArtists[i])
				}
			})
		}
		if len(a.homeAlbums) > 0 {
			a.section(c, "新专速递")
			a.albumGrid(c, a.homeAlbums)
		}
		if len(a.charts) > 0 {
			ui.Row(c).AlignItems(ui.Center).Margin(46, 0, 20, 0).Children(func() {
				ui.Text(c, "排行榜").FontSize(28).FontWeight(700).Grow(1)
				filterButton(c, "查看全部榜单", false).OnClick(func() { a.open(route{page: pageDiscover, query: "排行榜"}) })
			})
			a.playlistGrid(c.Key("charts"), a.charts[:min(5, len(a.charts))])
		}

	})
}

func weekday(t time.Time) string {
	return [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[t.Weekday()]
}

func (a *app) chartsPage(c *ui.Context) {
	loading := len(a.charts) == 0
	if loading && !a.homeLoading {
		a.centered(c, a.homeErr, a.loadHome)
		return
	}
	top, bottom := a.insets(c)
	ui.Scroll(c.Key("charts")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		a.heading(c, "排行榜", "每日更新")
		if loading {
			s := a.pageShimmer(c)
			a.chartCardsSkeleton(c, s)
			a.section(c, "全部榜单")
			a.playlistCardsSkeleton(c, s, a.columns(c, 150)*2, 150, true)
			return
		}

		a.chartSections(c)
	})
}

// chartSections is shared by the discovery filter and the standalone charts page.
func (a *app) chartSections(c *ui.Context) {
	if len(a.charts) == 0 {
		return
	}
	w, _ := c.Size()
	cols := 2
	if w < 800 {
		cols = 1
	}
	n := min(4, len(a.charts))
	a.section(c, "榜单速览").Margin(0, 0, 18, 0)
	ui.Grid(c.Key("featured-charts")).Columns(cols).Gap(16).Children(func() {
		for i := range a.charts[:n] {
			a.chartCard(c.Key(a.charts[i].ID), &a.charts[i])
		}
	})
	if len(a.charts) > n {
		a.section(c, "更多榜单")
		a.playlistGrid(c.Key("more-charts"), a.charts[n:])
	}
}

// playlistCard is a square cover with the playlist's name below: it lifts
// under the pointer, where a play button shows on it.
func (a *app) playlistCard(c *ui.Context, pl *netease.Playlist) {
	if a.coverCard(c, pl.Name, pl.Cover(), pl.PlayCount, pl.UpdateFreq, func() { a.loadPlaylist(pl.ID, true) }) {
		a.open(route{page: pagePlaylist, id: pl.ID})
	}
}

// coverCard is a playlist's card, or a podcast's: its cover, with how many
// times it was played on it, over its name and a line under that. The
// button on the cover calls play; it reports whether the rest was clicked.
func (a *app) coverCard(c *ui.Context, name, cover string, playCount float64, sub string, play func(), menu ...func(*ui.Menu)) bool {
	p := a.pal
	// From the top: a grid stretches the cards of a row as tall as the
	// tallest, whose name or line under it takes two lines.
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label(name)
	if len(menu) > 0 {
		card.ContextMenu(menu[0])
	}
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(10)
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -3}
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(lift, ui.Snappy)
		art.Children(func() {
			a.coverFill(c, cover, 400, 10)
			a.coverPlayButton(c, hovered, play)
		})
		if playCount > 0 && (a.route.page == pageDiscover || a.route.page == pageCharts) {
			ui.Text(c, "▶ "+plays(playCount)).FontSize(11).TextColor(p.muted)
		}
		ui.Text(c, name).FontSize(15).FontWeight(650).MaxLines(2).LineHeight(1.35)
		if sub != "" {
			ui.Text(c, sub).FontSize(12).TextColor(p.muted).SingleLine().Margin(-6, 0, 0, 0)
		}
	})
	return card.Clicked()
}

// coverFill fills its parent with a cover, its corners rounded by radius.
// The cover rounds itself rather than its parent clipping it: a clip moved
// by a transform, as a card lifting under the pointer, is drawn on the CPU.
func (a *app) coverFill(c *ui.Context, url string, px int, radius float32) {
	bmp := bitmap(a.art.get(netease.Sized(url, px), false))
	box := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(radius)
	in := artIn(box, bmp)
	if in < 1 {
		box.Background(a.pal.hover)
	}
	box.Children(func() { artImage(c, bmp, radius, in) })
}

// coverPlayButton plays a playlist or a podcast from its card, without
// opening it.
func (a *app) coverPlayButton(c *ui.Context, shown bool, play func()) {
	b := ui.ButtonBase(c.Key("play")).Absolute().Size(38, 38).Radius(999).Center().Background(ui.RGBA(24, 24, 26, .65)).TextColor(white).Label("播放").FocusRing(false).Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, .3))
	if a.route.page == pageHome || a.route.page == pageDiscover {
		b.TopPercent(50).LeftPercent(50).Margin(-23, 0, 0, -23).Size(46, 46)
	} else {
		b.Bottom(10).Right(10)
	}
	pose := ui.Pose{Fade: 1, Scale: 0.8, Y: 6}
	if shown || b.Hovered() {
		pose = ui.Pose{Scale: 1}
		if b.Hovered() {
			pose.Scale = 1.08
		}
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() { ui.Icon(c, iconPlay).Size(15, 15).Margin(0, 0, 0, 2) })
	if b.Clicked() {
		play()
	}
}

// chartCard shows a chart with its top three songs beside its cover.
func (a *app) chartCard(c *ui.Context, pl *netease.Playlist) {
	p := a.pal
	card := ui.ButtonBase(c).Gap(18).Padding(16).Radius(14).AlignItems(ui.Center).FocusRing(false).Label(pl.Name)
	hovered := card.Hovered()
	bg := p.hover
	if hovered {
		bg = p.press
	}
	card.Background(card.AnimateColor("bg", bg, ui.Animation{Duration: 140 * time.Millisecond}))
	card.Children(func() {
		ui.Box(c).Size(100, 100).Radius(9).Shadow(0, 3, 10, -2, p.cardShade).Children(func() {
			a.coverFill(c, pl.Cover(), 200, 9)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(5).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, pl.Name).FontSize(18).FontWeight(700).SingleLine()
				ui.Text(c, pl.UpdateFreq).FontSize(11).TextColor(p.faint).SingleLine()
			})
			for i, line := range pl.Preview {
				ui.Row(c).Gap(7).Children(func() {
					ui.Textf(c, "%d", i+1).FontSize(12).FontWeight(700).Width(10).FontFeatures("tnum").TextColor(p.muted)
					ui.RichText(c,
						ui.Span{Text: line.First, Weight: 500},
						ui.Span{Text: "  " + line.Second, Color: p.muted},
					).FontSize(12).SingleLine().Shrink(1).MinWidth(0)
				})
			}
		})
	})
	if card.Clicked() {
		a.open(route{page: pagePlaylist, id: pl.ID})
	}
}

func (a *app) playlistPage(c *ui.Context) {
	id := a.route.id
	pl := a.playlists[id]
	if pl == nil {
		if !a.loadingPl[id] {
			a.centered(c, a.plErr[id], func() { a.loadPlaylist(id, false) })
			return
		}
		s := a.pageShimmer(c)
		a.trackSkeleton(c, s, true, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		return
	}
	a.trackRows(c, pl.Tracks, true, func(c *ui.Context) { a.playlistHeader(c, pl) })
}

func (a *app) playlistHeader(c *ui.Context, pl *netease.Playlist) {
	p := a.pal
	ui.Row(c).Gap(48).AlignItems(ui.Center).Padding(0, 0, 52, 0).Children(func() {
		a.playableDetailCover(c, pl.Cover(), pl.Tracks)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			kind := "歌单"
			if pl.UpdateFreq != "" {
				kind = "榜单 · " + pl.UpdateFreq
			}
			ui.Text(c, kind).FontSize(12).FontWeight(600).TextColor(p.accent)
			ui.Text(c, pl.Name).FontSize(36).FontWeight(700).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			var meta []string
			if pl.Creator.Nickname != "" {
				ui.RichText(c).FontSize(16).FontWeight(600).Children(func() {
					if pl.Creator.UserID > 0 {
						a.link(c, pl.Creator.Nickname, p.text, p.accent, route{page: pageUser, id: pl.Creator.UserID})
					} else {
						ui.Text(c, pl.Creator.Nickname)
					}
				})
			}
			if pl.UpdateTime > 0 {
				meta = append(meta, time.UnixMilli(pl.UpdateTime).Format("2006.01.02")+" 更新")
			}
			meta = append(meta, fmt.Sprintf("%d 首", max(pl.TrackCount, len(pl.Tracks))))
			if pl.PlayCount > 0 {
				meta = append(meta, plays(pl.PlayCount)+" 次播放")
			}
			ui.Text(c, strings.Join(meta, " · ")).FontSize(13).TextColor(p.muted)
			if d := strings.TrimSpace(pl.Description); d != "" {
				a.detailDescription(c, d)
			}
			ui.Row(c).Wrap().Gap(10).Margin(24, 0, 0, 0).Children(func() {
				if pillButton(c.Key("play"), iconPlay, "播放", p.accent, p.onAccent).Clicked() {
					a.playDetail(pl.Tracks, false)
				}
				if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
					a.playDetail(pl.Tracks, true)
				}
				if mine := a.editable(pl.ID); mine != nil {
					more := iconButton(c.Key("more"), iconMore, 18, p.muted, "更多")
					more.Menu(func(m *ui.Menu) {
						if m.Item("重命名…").Chosen() {
							a.startRename(mine)
						}
						m.Separator()
						if m.Item("删除歌单…").Chosen() {
							a.confirmDelete(mine)
						}
					})
				}
				switch {
				case pl.SpecialType != 5:
				case a.pl.Radio() == heartRadio:
					if pillButton(c.Key("heart"), iconX, "退出心动模式", p.hover.Over(p.bg), p.text).Clicked() {
						a.exitRadio()
					}
				default:
					label := "心动模式"
					if a.radioStarting == heartRadio {
						label = "正在开启…"
					}
					if pillButton(c.Key("heart"), iconHeartFill, label, p.hover.Over(p.bg), p.text).Clicked() {
						a.startHeartMode()
					}
				}
			})
		})
	})
}

// trackRows lists tracks under a header, which scrolls with them; a click
// plays the list from the track clicked.
func (a *app) trackRows(c *ui.Context, tracks []netease.Track, covers bool, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	p := a.pal
	cur := a.pl.Current()
	// A song's menu can take it out of the playlist it is listed in.
	var pid int64
	if a.route.page == pagePlaylist {
		pid = a.route.id
	}
	w, _ := c.Size()
	// An album's page has no column of the album its songs are on.
	wide := w > 820 && a.route.page != pageAlbum
	top, bottom := a.insets(c)
	// Rows out of view are not built, their marks with them: a list
	// scrolled away from its start or end has content under the bars.
	a.underTop, a.underBottom = true, true
	rows := len(tracks) + 1 + len(footer)
	ui.List(c.Key(a.route), &a.trackList, rows, func(i int) {
		if i == 0 {
			ui.Column(c).Children(func() {
				a.startMark(c)
				header(c)
			})
			return
		}
		if i == rows-1 {
			defer a.endMark(c)
		}
		if i > len(tracks) {
			footer[i-len(tracks)-1](c)
			return
		}
		i--
		tr := &tracks[i]
		playing := cur != nil && cur.ID == tr.ID
		row := ui.ButtonBase(c.Key(tr.ID)).Height(64).PaddingX(12).Gap(14).Radius(10).AlignItems(ui.Center).FocusRing(false).Label(tr.Name).
			ContextMenu(a.songItems(tr, pid))
		hovered := row.Hovered()
		bg := ui.Transparent
		switch {
		case playing:
			bg = p.primaryBG()
		case hovered:
			bg = p.hover
		}
		row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
		if !tr.Playable() {
			row.Opacity(0.42)
		}
		row.Children(func() {
			ui.Box(c).Width(26).Center().Children(func() {
				switch {
				case playing:
					a.nowPlayingBars(c)
				case hovered && tr.Playable():
					ui.Icon(c, iconPlay).Size(13, 13).TextColor(p.text)
				default:
					ui.Textf(c, "%d", i+1).FontSize(13).FontFeatures("tnum").TextColor(p.faint)
				}
			})
			if covers {
				a.cover(c, tr.Album.PicURL, 100, 46, 8)
			}
			ui.Column(c).Grow(3).Basis(0).MinWidth(0).Gap(3).Children(func() {
				name := p.text
				if playing {
					name = p.accent
				}
				spans := []ui.Span{{Text: tr.Name, Weight: 600, Color: name}}
				if len(tr.Alias) > 0 {
					spans = append(spans, ui.Span{Text: "（" + tr.Alias[0] + "）", Color: p.faint})
				}
				ui.RichText(c, spans...).FontSize(17).SingleLine()
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if !tr.Playable() {
						ui.Text(c, "VIP").FontSize(9).FontWeight(800).TextColor(p.accent).Padding(0, 4).Radius(3).Border(1, p.accent.Alpha(0.6))
					}
					a.artistLinks(c, tr, p.muted, p.text).FontSize(12.5).SingleLine().Shrink(1).MinWidth(0)
					if tr.Reason != "" {
						ui.Text(c, tr.Reason).FontSize(10.5).FontWeight(600).TextColor(p.accent).Padding(1, 6).Radius(999).Background(p.accent.Alpha(0.1)).SingleLine()
					}
				})
			})
			if wide {
				album := ui.RichText(c).Grow(2).Basis(0).MinWidth(0).FontSize(12.5).TextColor(p.muted).SingleLine()
				album.Children(func() {
					if tr.Album.ID == 0 {
						ui.Text(c, tr.Album.Name)
					} else {
						a.link(c.Key("album"), tr.Album.Name, p.muted, p.text, route{page: pageAlbum, id: tr.Album.ID})
					}
				})
			}
			// A liked song's heart shows; the others' as the pointer is on them.
			ui.Box(c).Size(32, 32).Center().Children(func() {
				if a.liked[tr.ID] || hovered {
					a.likeButton(c.Key("like"), tr, 15, p.muted)
				}
			})
			ui.Text(c, clock(tr.Length())).FontSize(12.5).FontFeatures("tnum").TextColor(p.faint).Width(44).TextAlign(ui.End)
		})
		if row.Clicked() && tr.Playable() {
			a.pl.PlayQueue(tracks, i)
		}
	}).Grow(1).Padding(top+pageGap, pagePad-12, bottom+32, pagePad-12).ScrollbarInsets(top, 0, bottom, 0)
}

// nowPlayingBars dance in the row of the track playing.
func (a *app) nowPlayingBars(c *ui.Context) {
	color := a.pal.accent
	ui.Box(c).Size(16, 14).Draw(func(p *ui.Painter, r ui.Rect) {
		lv, live := a.pl.Levels()
		a.rowBars.step(p.Now(), lv, live)
		drawBars(p, r, &a.rowBars, 4, color, 2.5)
		if live || a.rowBars.group(0, player.Bands) > 0.01 {
			p.AnimationFrame()
		}
	})
}

func detailCoverSize(c *ui.Context) float32 {
	w, _ := c.Size()
	return min(288, max(156, w*.24))
}
func detailTitleSize(c *ui.Context) float32 {
	w, _ := c.Size()
	return min(56, max(34, w*.048))
}
func (a *app) detailDescription(c *ui.Context, description string) {
	text := ui.Text(c, description).FontSize(14).TextColor(a.pal.muted).LineHeight(1.45)
	if !a.descriptionOpen {
		text.MaxLines(3)
	}
	if len([]rune(description)) > 64 || strings.Count(description, "\n") >= 3 {
		label := "展开简介"
		if a.descriptionOpen {
			label = "收起简介"
		}
		ui.ButtonBase(c).Label(label).AlignSelf(ui.Start).Children(func() { ui.Text(c, label).FontSize(13).TextColor(a.pal.accent) }).OnClick(func() { a.descriptionOpen = !a.descriptionOpen })
	}
}

func (a *app) playDetail(tracks []netease.Track, shuffle bool) {
	if len(tracks) == 0 {
		return
	}
	if a.pl.Shuffle() != shuffle {
		a.pl.ToggleShuffle()
	}
	a.pl.PlayQueue(tracks, firstPlayable(tracks))
}

func (a *app) playableDetailCover(c *ui.Context, url string, tracks []netease.Track) {
	size := detailCoverSize(c)
	b := ui.ButtonBase(c.Key("detail-cover")).Size(size, size).Radius(14).Center().Label("播放封面中的曲目").Disabled(len(tracks) == 0).Shadow(0, 18, 40, -10, a.pal.cardShade)
	hovered := b.Hovered()
	b.Children(func() {
		a.coverFill(c, url, 600, 14)
		if hovered {
			ui.Box(c).Size(52, 52).Radius(26).Center().Background(ui.RGBA(24, 24, 26, .6)).Children(func() { ui.Icon(c, iconPlay).Size(20, 20).TextColor(white) })
		}
	})
	b.OnClick(func() { a.playDetail(tracks, false) })
}
