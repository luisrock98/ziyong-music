# 构建与发布

本文件适用于「自用MUSIC」衍生版本。应用没有配置自动更新源、Apple Developer ID 签名或公证；构建无需原作者的证书、Apple 账号或更新签名密钥。

## 分享源码

保留 `LICENSE`、README 中的上游出处与 Git 历史。发布前运行：

```bash
go test ./internal/...
CGO_ENABLED=0 go test ./...
go tool mygo vet .
git diff --check
```

检查提交列表，确认不包含 `.mygo/`、`build/`、登录态、歌曲缓存或本机凭据。应用的版本号由 `mygo.json` 的 `version` 决定，打包工具会将它写入可执行文件和安装包。

## 在本机打包

在 macOS 上生成当前架构的应用：

```bash
go tool mygo build -skip-dmg
```

产物位于 `build/darwin-arm64/` 或 `build/darwin-amd64/`。需要磁盘映像时去掉 `-skip-dmg`；同时构建两种 Mac 架构可使用：

```bash
go tool mygo build -platform darwin/arm64,darwin/amd64
```

Windows 和 Linux 的构建目标也予以保留，但需在对应系统验证后再宣称支持：

```bash
go tool mygo build -platform windows/amd64,windows/arm64
go tool mygo build -platform linux/amd64,linux/arm64
```

从 macOS 或 Linux 构建 Windows 安装程序需要先安装 NSIS。macOS 的磁盘映像与签名步骤需要在 macOS 上执行。

## GitHub Actions 构建

1. 将待构建提交推送到本仓库。
2. 打开 **Actions → Build packages → Run workflow**，选择要构建的分支。
3. 等待测试和三个平台的构建全部完成，在该次运行的 **Artifacts** 下载各平台安装包。构建产物保留 14 天。
4. 解压 Artifacts 后，在对应系统检查安装、启动、扫码登录、播放和退出。macOS 包仅有临时签名，没有 Apple 公证。

此工作流只生成安装包，不会创建或发布 Release，也不会连接上游的更新服务。`CI` 工作流在 `main` 推送或 PR 时运行测试与 MyGo 检查。

## 发布可下载版本

1. 修改 `mygo.json` 的 `version`，同步更新 README 的当前版本，并在 `CHANGELOG.md` 顶部记录改动。
2. 提交并推送代码，确认 CI 通过；针对同一提交运行 **Build packages**，下载并实机检查安装包。
3. 在 GitHub 的 **Releases → Draft a new release** 中，以该提交创建 `v<version>` 标签，填写变更说明和已验证平台。
4. 上传对应的 `.dmg`、`*-setup-*.exe`、`.deb` 或 `.tar.gz` 文件。说明 macOS 的签名状态，以及本版本需要手动下载更新。
5. 确认标签对应安装包的源码提交后再发布。保留 GitHub 自动提供的源码下载入口。

当前配置使用手动上传流程。`go tool mygo build -upload` 需要另行配置 `updates.github`。

## 播客转写依赖

语音模型和运行库在首次使用时下载，不随客户端安装包发布。`internal/speech/natives.go` 保留了上游的运行库下载地址和校验值。若改为自行托管，先调整 `scripts/sherpa/main.go` 的目标仓库，再生成运行库与校验清单，并验证下载、校验、加载和转写；不要只替换 URL。
