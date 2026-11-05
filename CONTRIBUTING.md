# 参与贡献

欢迎提交 Issue 或 Pull Request。这个仓库维护「自用MUSIC」个人衍生版本，原项目及贡献者的出处见 [README](README.md)。

## 报告问题

请提供应用版本、操作系统及芯片架构、复现步骤、期望结果和实际结果。界面问题可以附截图；提交前请遮住账号、昵称、私人歌单和其他个人信息，不要附登录 Cookie、二维码、凭据文件或歌曲缓存。

## 本地开发

安装 `go.mod` 指定版本或更新的 Go，在仓库根目录执行：

```bash
go mod download
go tool mygo dev
```

修改后运行：

```bash
go test ./internal/...
CGO_ENABLED=0 go test ./...
go tool mygo vet .
```

遵循 [AGENTS.md](AGENTS.md) 中的测试约定：优先验证输入产生的行为、独立逻辑和修复的回归，不为界面文案、默认值或框架自身行为添加测试。新 UI 测试需使用 `-count=5` 检查稳定性；需要截图时使用现有的 `NM_FRAMES` 测试。联网测试只有显式设置 `LIVE` 才运行。

一个 PR 尽量解决一个问题，说明行为变化、验证方法和平台限制。涉及个人数据的日志请先脱敏。本项目沿用 [AGPL-3.0](LICENSE)，贡献代码时请保留相应的来源与许可说明。
