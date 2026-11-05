# 自用MUSIC

用 Go 和 [MyGo](https://github.com/egoist/mygo) 写的第三方网易云音乐桌面客户端，原生界面，无 WebView。

[项目仓库](https://github.com/luisrock98/ziyong-music) · [问题反馈](https://github.com/luisrock98/ziyong-music/issues) · [构建工作流](https://github.com/luisrock98/ziyong-music/actions)

这是基于 [egoist/mygo-cloudmusic](https://github.com/egoist/mygo-cloudmusic) 的个人衍生版本，代码起点为 [Amamiya23/mygo-qyin 中的网易云版本 `5703a7a`](https://github.com/Amamiya23/mygo-qyin/tree/5703a7a90a8e502d4de185855970620228a58679)，保留原项目的提交历史与 AGPL-3.0 许可证。

本版本使用独立的应用名称和标识，不迁移原版的用户数据，也不连接原版的自动更新源。2026-11-05 整理为开源分享版本；版本号沿用上游的 `0.1.20`。

## 功能

- 扫码登录，打包版本的登录态保存在系统凭据存储中
- 每日推荐、推荐歌单、排行榜、我的歌单
- 搜索单曲、歌手、专辑、用户与播客，输入时显示搜索建议，保留搜索历史
- 歌手页的热门与最新歌曲、专辑与动态，关注歌手；专辑页与用户页
- 播客：浏览与收听节目，订阅播客；在本机把节目转写成字幕（Qwen3-ASR），像歌词一样随播放滚动
- 私人 FM（推荐 / 熟悉 / 探索）与心动模式
- 边下边播，支持标准 / 较高 / 极高 / 无损（FLAC）/ Hi-Res 音质
- 听过的歌曲缓存在本机，再次播放无需等待
- 音量均衡，暂停与继续时淡出淡入，拖动进度无杂音
- 退出后再打开，从上次的播放队列与进度继续
- 逐字歌词与翻译，正在播放页的「律动」视图与歌曲评论
- 刘海播放器：在屏幕顶部刘海处显示迷你播放器，悬停展开（macOS，在设置中开启）
- 喜欢歌曲、播放队列、随机与循环播放
- 浅色 / 深色 / 跟随系统

## 使用与平台

当前以源码分享为主，已在 Apple Silicon Mac 上验证构建与测试。macOS 最低版本为 13；代码保留 Windows 和 Linux 支持，这两个平台尚未在本衍生版本中进行实机验证。

打开应用后，点左下角「登录」，用手机网易云音乐扫码确认即可进入自己的歌单。歌曲能否播放、可用音质以及会员内容由网易云账号权限与服务端决定。

本版本没有配置 Apple Developer ID 签名或公证，macOS 构建使用临时签名；不提供应用内自动更新。

## 从源码构建

安装 Go **1.27.3 或更高版本**（以 [go.mod](go.mod) 为准）和 Git，然后执行：

```bash
git clone https://github.com/luisrock98/ziyong-music.git
cd ziyong-music
go mod download
go tool mygo dev
```

打包为可双击的 macOS 应用：

```bash
go tool mygo build -skip-dmg
```

Apple Silicon 的产物为 `build/darwin-arm64/自用MUSIC.app`，Intel Mac 为 `build/darwin-amd64/自用MUSIC.app`。桌面客户端不需要 Node.js 或 Bun，也不需要单独部署 API 服务。

仓库附带手动构建工作流，可在 GitHub Actions 中运行 **Build packages** 获取测试用安装包。构建和发布步骤见 [RELEASING.md](RELEASING.md)。

## 测试与贡献

```bash
go test ./internal/...
CGO_ENABLED=0 go test ./...
go tool mygo vet .
```

普通测试不需要网易云账号；联网测试仅在设置 `LIVE` 时运行。欢迎提交问题和改进，复现信息与测试约定见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 数据与外部服务

- 打包版本使用系统凭据存储保存登录态；开发模式把登录态保存在应用用户数据目录下的私有 `session` 文件中。
- 设置、播放队列和歌曲缓存保存在本机应用数据或缓存目录中，不应上传到仓库。`build/`、`.mygo/` 和本地凭据文件已列入忽略规则。
- 登录、搜索和播放会访问网易云服务。播客转写在本机执行，首次使用需下载约 1 GB 模型及运行库；下载来源包括 ModelScope、Hugging Face 和上游 GitHub Releases。
- `website/` 是保留的上游展示站源码，仍使用上游品牌、下载链接和部署配置，不是本版本的下载入口；见 [网站说明](website/README.md)。

## 致谢

- [egoist/mygo-cloudmusic](https://github.com/egoist/mygo-cloudmusic) 及其贡献者：桌面客户端的原始实现
- [Amamiya23/mygo-qyin](https://github.com/Amamiya23/mygo-qyin)：本地版本的代码来源
- [MyGo](https://github.com/egoist/mygo)：原生 UI 与应用框架
- [NeteaseCloudMusicApi](https://github.com/Binaryify/NeteaseCloudMusicApi)：eapi 加密与登录流程的参考
- [folia-major](https://github.com/chthollyphile/folia-major)：歌词动画效果的参考
- [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx)、[Qwen3-ASR](https://github.com/QwenLM/Qwen3-ASR) 与 [Silero VAD](https://github.com/snakers4/silero-vad)：本机语音识别

本项目与网易云音乐官方无关，音乐版权归各权利人所有。

## 许可证

[AGPL-3.0](LICENSE)
