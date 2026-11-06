package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// albumDetail loads an album from NetEase; tests replace it.
var albumDetail = netease.AlbumDetail

// albumData is what the app has of an album.
type albumData struct {
	album   *netease.AlbumProfile
	loading bool
	err     string
}

func (a *app) loadAlbum(id int64) {
	d := a.albums[id]
	if d == nil {
		d = &albumData{}
		a.albums[id] = d
	}
	if d.album != nil || d.loading {
		return
	}
	d.loading, d.err = true, ""
	go func() {
		al, err := albumDetail(id)
		a.post(func() {
			d.loading = false
			if err != nil {
				d.err = "专辑加载失败"
				return
			}
			d.album = al
		})
	}()
}

func (a *app) albumPage(c *ui.Context) {
	id := a.route.id
	d := a.albums[id]
	if d.album == nil {
		if !d.loading {
			a.centered(c, d.err, func() { a.loadAlbum(id) })
			return
		}
		s := a.pageShimmer(c)
		a.trackSkeleton(c, s, false, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		return
	}
	a.trackRows(c, d.album.Songs, false, func(c *ui.Context) { a.albumHeader(c, d.album) })
}

func (a *app) albumHeader(c *ui.Context, al *netease.AlbumProfile) {
	p := a.pal
	ui.Row(c).Gap(48).AlignItems(ui.Center).Padding(0, 0, 52, 0).Children(func() {
		a.playableDetailCover(c, al.PicURL, al.Songs)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, "专辑").FontSize(12).FontWeight(600).TextColor(p.accent)
			ui.Text(c, al.Name).FontSize(detailTitleSize(c)).FontWeight(700).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			if len(al.Artists) > 0 {
				ui.RichText(c).FontSize(17).FontWeight(600).Children(func() {
					for i, ar := range al.Artists {
						if i > 0 {
							ui.Text(c, " / ")
						}
						if ar.ID != 0 {
							a.link(c.Key(i), ar.Name, p.text, p.accent, route{page: pageArtist, id: ar.ID})
						} else {
							ui.Text(c, ar.Name)
						}
					}
				})
			}
			meta := []string{}
			if al.PublishTime > 0 {
				meta = append(meta, time.UnixMilli(al.PublishTime).Format("2006.01.02"))
			}
			meta = append(meta, fmt.Sprintf("%d 首", len(al.Songs)))

			ui.Text(c, strings.Join(meta, " · ")).FontSize(14).TextColor(p.muted)
			if d := strings.TrimSpace(al.Description); d != "" {
				a.detailDescription(c, d)
			}
			ui.Row(c).Gap(10).Margin(24, 0, 0, 0).Children(func() {
				if pillButton(c.Key("play"), iconPlay, "播放", p.accent, p.onAccent).Clicked() {
					a.playDetail(al.Songs, false)
				}
				if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
					a.playDetail(al.Songs, true)
				}
			})
		})
	})
}

// albumGrid shows albums as cards, each opening its page.
func (a *app) albumGrid(c *ui.Context, albums []netease.AlbumProfile) {
	ui.Grid(c).Columns(a.columns(c, 156)).GapX(20).GapY(28).Children(func() {
		for i := range albums {
			a.albumCard(c.Key(albums[i].ID), &albums[i])
		}
	})
}

// albumCard is an album's cover, lifting under the pointer, over its name
// and the year it came out.
func (a *app) albumCard(c *ui.Context, al *netease.AlbumProfile) {
	p := a.pal
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(al.Name)
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
		art.Children(func() { a.coverFill(c, al.PicURL, 400, 10) })
		ui.Text(c, al.Name).FontSize(15).FontWeight(650).MaxLines(2).LineHeight(1.35)
		var meta []string
		if a.route.page != pageArtist {
			if names := al.ArtistNames(); names != "" {
				meta = append(meta, names)
			}
		}
		if al.PublishTime > 0 {
			meta = append(meta, time.UnixMilli(al.PublishTime).Format("2006"))
		}
		ui.Text(c, strings.Join(meta, " · ")).FontSize(12).TextColor(p.muted).SingleLine().Margin(-6, 0, 0, 0)
	})
	if card.Clicked() {
		a.open(route{page: pageAlbum, id: al.ID})
	}
}
