# 网站

> 此目录保留自上游 `egoist/mygo-cloudmusic`。品牌、下载地址、演示音频、统计脚本及 Cloudflare 配置仍属于上游展示站，尚未适配「自用MUSIC」。本次源码分享不部署此网站；桌面客户端的构建也不依赖它。若要部署自己的站点，需先替换这些配置与素材。

MyGo 云音乐的主页，用 TanStack Start 与 Solid 写成。构建时预渲染为静态文件，部署到 Cloudflare（Workers 静态资源，不运行 Worker 脚本）。

```bash
bun install          # 在仓库根目录
bun run dev          # http://localhost:3000
bun run build        # 输出到 website/dist/client
bun run deploy       # 构建并用 wrangler 部署
```
