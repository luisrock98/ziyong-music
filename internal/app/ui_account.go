package app

import (
	"runtime"
	"time"

	"github.com/egoist/mygo/ui"
	"rsc.io/qr"

	"netease-music-mygo/internal/netease"
)

// loginState is a QR code login under way.
type loginState struct {
	open     bool
	gen      int // counts logins, to stop the polls of closed ones
	key      string
	code     *qr.Code
	status   int // a QR state, or 0 while the code loads
	scanner  string
	avatar   string
	err      string
	finished bool
}

// boot signs in with the session in the keychain, if any, then loads the
// home page for whoever is listening.
func (a *app) boot() {
	a.homeLoading = true
	go func() {
		var prof *netease.Profile
		if netease.RestoreSession() {
			p, err := netease.Account()
			switch {
			case err == nil && p == nil:
				netease.ClearSession() // expired
			case err == nil:
				prof = p
			}
		}
		a.post(func() {
			a.signedIn(prof)
		})
	}()
}

// signedIn shows what is personal to prof, or to nobody.
func (a *app) signedIn(prof *netease.Profile) {
	a.profile = prof
	a.daily, a.homeArtists = nil, nil
	a.libraryFilter = 0
	a.libraryCollection = nil
	// Playback picks up where it was left, once the session that lets
	// the listener play their songs is back.
	a.restoreSession()
	a.mine, a.saved, a.recPlaylists, a.liked = nil, nil, nil, nil
	a.likedPrograms, a.askedPrograms = nil, nil
	// What the listener may play, and follows, changed: load the pages
	// again.
	a.playlists = map[int64]*netease.Playlist{}
	a.plErr = map[int64]string{}
	a.artists, a.albums, a.users, a.podcasts = map[int64]*artistData{}, map[int64]*albumData{}, map[int64]*userData{}, map[int64]*podcastData{}
	a.podcastTab = &podcastTab{}
	a.searches = map[route]*searchData{}
	a.load(a.route)
	a.loadHome()
	if prof == nil {
		return
	}
	a.loadLiked(prof.UserID)
	a.loadMine(prof.UserID)
}

func (a *app) openLogin() {
	a.login = loginState{open: true, gen: a.login.gen + 1}
	gen := a.login.gen
	go func() {
		key, err := netease.QRKey()
		var code *qr.Code
		if err == nil {
			code, err = qr.Encode(netease.LoginURL(key), qr.M)
		}
		a.post(func() {
			if gen != a.login.gen || !a.login.open {
				return
			}
			if err != nil {
				a.login.err = "无法获取二维码，请检查网络"
				return
			}
			a.login.key, a.login.code, a.login.status = key, code, netease.QRWaiting
			a.pollLogin(gen, key)
		})
	}()
}

// pollLogin asks every two seconds whether the code was scanned, until the
// login ends or the dialog closes.
func (a *app) pollLogin(gen int, key string) {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			code, msg, name, avatar, err := netease.QRCheck(key)
			var prof *netease.Profile
			if code == netease.QRConfirmed {
				netease.SaveSession()
				prof, _ = netease.Account()
			}
			done := make(chan bool, 1)
			a.post(func() {
				l := &a.login
				if gen != l.gen || !l.open {
					done <- true
					return
				}
				switch {
				case err != nil:
					// Try again: the network may come back.
				case code == netease.QRConfirmed:
					l.status, l.finished = code, true
					a.signedIn(prof)
					if prof != nil {
						a.pickUp(prof.UserID)
					}
					a.closeLoginSoon(gen)
				case code == netease.QRExpired || code == netease.QRWaiting || code == netease.QRScanned:
					l.status = code
					if name != "" {
						l.scanner, l.avatar = name, avatar
					}
				default:
					l.err = msg
					if l.err == "" {
						l.err = "登录失败，请稍后再试"
					}
				}
				done <- l.status == netease.QRExpired || l.finished || l.err != ""
			})
			if <-done {
				return
			}
		}
	}()
}

// closeLoginSoon closes the dialog a moment after it showed the welcome.
func (a *app) closeLoginSoon(gen int) {
	go func() {
		time.Sleep(1200 * time.Millisecond)
		a.post(func() {
			if gen == a.login.gen {
				a.login.open = false
			}
		})
	}()
}

func (a *app) signOut() {
	a.login.gen++
	a.signedIn(nil)
	go netease.Logout()
}

// loginDialog shows the QR code to scan with the NetEase app on a phone.
func (a *app) loginDialog(c *ui.Context) {
	p := a.pal
	l := &a.login
	ui.DialogBase(c.Key("login"), &l.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(368).Padding(32, 28, 24, 28).Radius(18).Gap(8).AlignItems(ui.Center).
			Background(p.bg).Border(1, p.hairline).Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.4)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})
		title, sub := "扫码登录", "打开网易云音乐 App，扫一扫登录"
		switch {
		case l.finished:
			title, sub = "登录成功", "欢迎回来，"+a.profileName()
		case l.err != "":
			sub = l.err
		case l.status == netease.QRScanned:
			title, sub = "扫描成功", "请在手机上确认登录"
		case l.status == netease.QRExpired:
			sub = "二维码已失效"
		}
		ui.Text(c, title).FontSize(30).FontWeight(750)
		ui.Text(c, sub).FontSize(13).TextColor(p.muted).TextAlign(ui.Center).Margin(0, 0, 14, 0)

		ui.Box(c).Size(224, 224).Radius(16).Background(ui.RGB(255, 255, 255)).Border(1, p.hairline).Children(func() {
			if l.code != nil {
				code := l.code
				opacity := float32(1)
				if l.status == netease.QRScanned || l.status == netease.QRExpired || l.finished || l.err != "" {
					opacity = 0.12
				}
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Opacity(opacity).
					Draw(func(pt *ui.Painter, r ui.Rect) { drawQR(pt, r, code, 14) })
			} else if l.err == "" {
				// On white whatever the appearance, as the code is.
				s := shimmer{base: ui.RGBA(0, 0, 0, 0.08), sheen: ui.RGBA(0, 0, 0, 0.03)}.in(c)
				qrSkeleton(c, s, 14).Absolute().Top(0).Left(0).Right(0).Bottom(0)
			}
			ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(10).Children(func() {
				switch {
				case l.finished || l.status == netease.QRScanned:
					if bmp := bitmap(a.art.get(netease.Sized(l.avatar, 160), false)); bmp != nil {
						pic := ui.Box(c).Size(72, 72).Radius(36).Shadow(0, 6, 16, -4, ui.RGBA(0, 0, 0, 0.3))
						in := artIn(pic, bmp)
						pic.Children(func() { artImage(c, bmp, 36, in) })
					}
					if l.scanner != "" {
						ui.Text(c, l.scanner).FontSize(14).FontWeight(600).TextColor(ui.Hex("#1d1d1f"))
					}
				case l.status == netease.QRExpired || l.err != "":
					if pillButton(c.Key("refresh"), iconRepeat, "刷新二维码", p.accent, p.onAccent).Clicked() {
						a.openLogin()
					}
				}
			})
		})
		if !l.finished {
			filterButton(c.Key("cancel"), "取消", false).Margin(12, 0, 0, 0).OnClick(func() { l.open = false; l.gen++ })
		}
	})
}

// drawQR draws a QR code in r, as crisp squares on white, inside a quiet
// zone of pad.
func drawQR(p *ui.Painter, r ui.Rect, code *qr.Code, pad float32) {
	n := code.Size
	cell := (min(r.W, r.H) - 2*pad) / float32(n)
	x0 := r.X + (r.W-cell*float32(n))/2
	y0 := r.Y + (r.H-cell*float32(n))/2
	ink := ui.Hex("#141416")
	for y := range n {
		for x := range n {
			if code.Black(x, y) {
				// A hair larger, so neighbors join without seams.
				p.Fill(ui.Rect{X: x0 + float32(x)*cell, Y: y0 + float32(y)*cell, W: cell + 0.3, H: cell + 0.3}, ink, 0)
			}
		}
	}
}

func (a *app) profileName() string {
	if a.profile == nil {
		return ""
	}
	return a.profile.Nickname
}

// accountRow opens the account menu from the avatar in the navigation bar.
func (a *app) accountRow(c *ui.Context) {
	p := a.pal
	b := ui.ButtonBase(c.Key("account")).Size(38, 38).Radius(19).Center()
	if b.Hovered() || a.accountOpen {
		b.Background(p.hover)
	}
	if a.profile == nil {
		a.accountOpen = false
		b.Label("登录").Children(func() { ui.Icon(c, iconUser).Size(19, 19).TextColor(p.muted) })
		b.OnClick(a.openLogin)
		return
	}
	b.Label(a.profile.Nickname).Expanded(a.accountOpen).Children(func() { a.avatar(c, a.profile.AvatarURL, 30, p.press) })
	if b.Clicked() {
		a.accountOpen = !a.accountOpen
	}
	a.accountMenu(c, b)
}

func (a *app) accountMenu(c *ui.Context, row ui.Element) {
	pop := popoverColors(a.pal.dark)
	ui.PopoverBase(c.Key("account-menu"), row, &a.accountOpen, func(panel ui.Element) {
		panel.AttachTo(row, ui.AnchorBottomEnd, ui.AnchorTopEnd).Margin(6, 0, 0, 0).Padding(5).Gap(1).
			Radius(12).Background(pop.fill).Border(0.5, pop.edge).Shadow(0, 8, 24, -4, ui.RGBA(0, 0, 0, 0.2)).
			Role(ui.RoleMenu).FocusGroup(ui.Vertical).Label("账户").TransformOrigin(0.5, 1).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.96, Y: 4}, Exit: &ui.Motion{Scale: 0.98}})
		panel.Width(194)
		shortcut := "Ctrl+,"
		if runtime.GOOS == "darwin" {
			shortcut = "⌘,"
		}
		if a.menuItem(c, iconUser, "我的主页", "").AutoFocus().Clicked() {
			a.accountOpen = false
			a.open(route{page: pageUser, id: a.profile.UserID})
		}
		if a.menuItem(c, iconSettings, "设置…", shortcut).Clicked() {
			a.accountOpen = false
			a.openSettings()
		}
		ui.Box(c).Height(0.5).Margin(4, 8).Background(pop.edge)
		if a.menuItem(c, iconSignOut, "退出登录", "").Clicked() {
			a.accountOpen = false
			a.signOut()
		}
	})
}

// menuItem is an item of a dropdown: an icon, its label, and the shortcut
// that does the same, if any.
func (a *app) menuItem(c *ui.Context, icon *ui.SVG, label, shortcut string) ui.Element {
	p := a.pal
	b := ui.ButtonBase(c.Key(label)).Role(ui.RoleMenuItem).Label(label).Height(30).PaddingX(8).Gap(9).Radius(7).
		AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	if b.Hovered() || b.FocusVisible() {
		b.Background(p.press)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(p.muted)
		ui.Text(c, label).FontSize(13).Grow(1)
		if shortcut != "" {
			ui.Text(c, shortcut).FontSize(12).TextColor(p.faint)
		}
	})
	return b
}

func (a *app) libraryPage(c *ui.Context) {
	a.scrollPage(c, func() {
		if a.profile == nil {
			a.heading(c, "音乐库", "")
			pillButton(c, iconUser, "登录网易云音乐", a.pal.accent, a.pal.onAccent).OnClick(a.openLogin)
			return
		}
		ui.Row(c).Gap(14).AlignItems(ui.Center).Margin(0, 0, 28, 0).Children(func() {
			a.cover(c, a.profile.AvatarURL, 100, 40, 20)
			ui.Text(c, a.profile.Nickname+"的音乐库").FontSize(34).FontWeight(750)
		})
		id := a.likedPlaylist()
		ui.Row(c).Gap(30).AlignItems(ui.Stretch).Children(func() {
			card := ui.ButtonBase(c.Key("liked-library")).Width(280).Height(224).Radius(16).Padding(24).Column().AlignItems(ui.Start).Justify(ui.End).Gradient(ui.Hex("#335eea"), ui.Hex("#684ac5"), 125).TextColor(white).Label("我喜欢的音乐")
			card.Children(func() {
				ui.Icon(c, iconHeartFill).Size(48, 48).Margin(0, 0, 30, 0)
				ui.Text(c, "我喜欢的音乐").FontSize(24).FontWeight(700)
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Textf(c, "%d 首歌曲", len(a.liked)).FontSize(14).Grow(1)
					iconButton(c.Key("play-liked"), iconPlay, 23, white, "播放喜欢的音乐").Disabled(id == 0 || a.loadingPl[id]).OnClick(func() { a.loadPlaylist(id, true) })
				})
			})
			card.OnClick(func() {
				if id != 0 {
					a.open(route{page: pagePlaylist, id: id})
				}
			})
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				if pl := a.playlists[id]; pl != nil {
					w, _ := c.Size()
					cols := 3
					if w < 1100 {
						cols = 2
					}
					a.compactTracks(c, pl.Tracks[:min(len(pl.Tracks), cols*4)], cols)
				} else if a.loadingPl[id] {
					ui.Text(c, "正在加载喜欢的歌曲…").TextColor(a.pal.muted)
				} else if a.plErr[id] != "" {
					ui.Button(c, "重新加载歌曲").OnClick(func() { a.loadPlaylist(id, false) })
				}
			})
		})
		ui.Row(c).Wrap().Gap(8).Margin(40, 0, 24, 0).AlignItems(ui.Center).Children(func() {
			for i, label := range []string{"全部歌单", "我的歌单", "收藏的歌单", "收藏的专辑", "关注的艺人"} {
				filterButton(c.Key(i), label, a.libraryFilter == i).OnClick(func() { a.libraryFilter = i; a.loadLibraryCollection() })
			}
			ui.Spacer(c)
			if a.libraryFilter < 3 {
				pillButton(c.Key("new"), iconPlus, "新建歌单", a.pal.hover, a.pal.text).OnClick(func() { a.newPlaylist(nil) })
			}
		})
		if a.libraryFilter >= 3 {
			a.collectionGrid(c)
			return
		}
		lists := append([]netease.Playlist{}, a.mine...)
		if a.libraryFilter == 2 {
			lists = nil
		}
		if a.libraryFilter != 1 {
			lists = append(lists, a.saved...)
		}
		ui.Grid(c).Columns(a.columns(c, 160)).GapX(24).GapY(40).Children(func() {
			for i := range lists {
				pl := &lists[i]
				if pl.SpecialType == 5 {
					continue
				}
				menu := func(m *ui.Menu) {}
				if mine := a.editable(pl.ID); mine != nil {
					menu = func(m *ui.Menu) {
						if m.Item("重命名…").Chosen() {
							a.startRename(mine)
						}
						m.Separator()
						if m.Item("删除歌单…").Chosen() {
							a.confirmDelete(mine)
						}
					}
				}
				if a.coverCard(c.Key(pl.ID), pl.Name, pl.Cover(), 0, pl.Creator.Nickname, func() { a.loadPlaylist(pl.ID, true) }, menu) {
					a.open(route{page: pagePlaylist, id: pl.ID})
				}
			}
		})
	})
}

type libraryCollection struct {
	albums                []netease.AlbumProfile
	artists               []netease.ArtistProfile
	albumPage, artistPage paged
}

func (a *app) loadLibraryCollection() {
	if a.profile == nil || a.libraryFilter < 3 {
		return
	}
	if a.libraryCollection == nil {
		a.libraryCollection = &libraryCollection{}
	}
	d := a.libraryCollection
	if a.libraryFilter == 3 {
		if !d.albumPage.loaded {
			a.moreLibraryAlbums(d)
		}
		return
	}
	if d.artistPage.loading || d.artistPage.loaded {
		return
	}
	d.artistPage = paged{loading: true}
	go func() {
		artists, err := followedArtists()
		a.post(func() {
			d.artistPage = paged{loaded: err == nil}
			if err != nil {
				d.artistPage.err = "关注的艺人加载失败"
				return
			}
			d.artists = artists
		})
	}()
}

func (a *app) moreLibraryAlbums(d *libraryCollection) {
	if d.albumPage.loading {
		return
	}
	d.albumPage.loading, d.albumPage.err = true, ""
	offset := len(d.albums)
	go func() {
		albums, more, err := netease.SavedAlbums(offset)
		a.post(func() {
			d.albumPage.loading = false
			if err != nil {
				d.albumPage.err = "收藏的专辑加载失败"
				return
			}
			d.albums = append(d.albums, albums...)
			d.albumPage.loaded, d.albumPage.more = true, more && len(albums) > 0
		})
	}()
}

func (a *app) collectionGrid(c *ui.Context) {
	d := a.libraryCollection
	if d == nil {
		return
	}
	if a.libraryFilter == 3 {
		a.albumGrid(c, d.albums)
		a.pageFoot(c, &d.albumPage, func() { a.moreLibraryAlbums(d) })
		if d.albumPage.loaded && len(d.albums) == 0 {
			a.empty(c, "还没有收藏专辑")
		}
		return
	}
	ui.Grid(c).Columns(a.columns(c, 160)).Gap(24).Children(func() {
		for i := range d.artists {
			a.artistCard(c.Key(d.artists[i].ID), &d.artists[i])
		}
	})
	a.pageFoot(c, &d.artistPage, a.loadLibraryCollection)
	if d.artistPage.loaded && len(d.artists) == 0 {
		a.empty(c, "还没有关注艺人")
	}
}
