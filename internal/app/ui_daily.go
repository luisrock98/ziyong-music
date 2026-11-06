package app

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
)

const dailyHeight = 198
const dailyRadius = 16

func (a *app) dailyCard(c *ui.Context, now time.Time) {
	card := ui.ButtonBase(c.Key("daily")).Height(198).Radius(16).Label("每日推荐").Background(ui.Hex("#415d72"))
	card.Children(func() {
		if len(a.daily) > 0 {
			a.coverFill(c, a.daily[0].Album.PicURL, 1000, 16)
		}
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(16).LinearGradient(ui.LinearGradient{Angle: 90, From: ui.RGBA(0, 0, 0, .35), To: ui.Transparent})
		ui.Column(c).Absolute().Top(28).Left(30).Children(func() {
			ui.Text(c, "每日").FontSize(48).FontWeight(800).TextColor(white).LineHeight(1.15)
			ui.Text(c, "推荐").FontSize(48).FontWeight(800).TextColor(white).LineHeight(1.15)
		})
		ui.Box(c).Absolute().Right(22).Bottom(20).Children(func() {
			ours, playing := a.dailyPlaying()
			button := playButton(c.Key("daily-play"), ours && playing, 24, white).Background(white.Alpha(.18)).Radius(999)
			button.OnClick(func() {
				if len(a.daily) == 0 && a.profile != nil {
					a.open(route{page: pageDaily})
					a.loadHome()
				} else if len(a.daily) == 0 {
					a.openLogin()
				} else if ours {
					a.pl.TogglePlay()
				} else {
					a.pl.PlayQueue(a.daily, firstPlayable(a.daily))
				}
			})
		})
	})
	card.OnClick(func() { a.open(route{page: pageDaily}) })

}

func (a *app) dailyPlaying() (ours, playing bool) {
	q, cur := a.pl.Queue(), a.pl.Current()
	if cur == nil || len(q) != len(a.daily) || len(q) == 0 || q[0].ID != a.daily[0].ID || q[len(q)-1].ID != a.daily[len(q)-1].ID {
		return false, false
	}
	return true, a.pl.Playing()
}

func (a *app) dailyBlurb() string {
	if a.profile == nil {
		return "大家都在听的好歌，登录后为你专属生成"
	}
	return "根据你的音乐口味生成，每天 6:00 更新"
}

func (a *app) dailyPage(c *ui.Context) {
	loading := len(a.daily) == 0
	if loading && !a.homeLoading {
		a.centered(c, "暂时没有推荐", a.loadHome)
		return
	}
	songs := a.daily
	s := a.pageShimmer(c)
	header := func(c *ui.Context) {
		p := a.pal
		w, h := c.Size()
		ui.Column(c).AlignItems(ui.Center).Padding(min(144, h*.14), 0, min(100, h*.1), 0).Children(func() {
			ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {

				title := []rune("每日歌曲推荐")
				spans := make([]ui.Span, len(title))
				for i, ch := range title {
					spans[i] = ui.Span{Text: string(ch), Color: ui.Hex("#ff512f").Mix(ui.Hex("#dd2476"), float32(i)/float32(len(title)-1)), Weight: 750}
				}
				ui.RichText(c, spans...).FontSize(min(84, w*.078)).LetterSpacing(3)
				if loading {
					s.text(c, 13, 1.3).Width(240)
					ui.Row(c).Gap(10).Margin(8, 0, 0, 0).Children(func() {
						s.bone(c, 18).Size(112, 36)
						s.bone(c, 18).Size(112, 36)
					})
					return
				}
				ui.Text(c, fmt.Sprintf("%s · %d 首", a.dailyBlurb(), len(songs))).FontSize(17).TextColor(p.muted).Margin(10, 0, 12, 0)
				ui.Row(c).Gap(10).Margin(8, 0, 0, 0).Children(func() {
					if pillButton(c.Key("play"), iconPlay, "播放全部", p.accent, p.onAccent).Clicked() {
						if a.pl.Shuffle() {
							a.pl.ToggleShuffle()
						}
						a.pl.PlayQueue(songs, firstPlayable(songs))
					}
					if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
						if !a.pl.Shuffle() {
							a.pl.ToggleShuffle()
						}
						a.pl.PlayQueue(songs, firstPlayable(songs))
					}
				})
			})
		})
	}
	if loading {
		a.trackSkeleton(c, s, true, header)
		return
	}
	a.trackRows(c, songs, true, header)
}
