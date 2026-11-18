# PicBerth

[English](README.en.md) | 简体中文

面向单一管理员自托管的个人图床：**一个可执行文件 + 一个本地数据目录**。静态内嵌 Svelte 5 管理界面，图片保存在本地磁盘，SQLite 保存元数据；编译后的二进制文件无需 Node.js 运行时，也没有外部数据库、Redis 缓存或对象存储依赖。

> **说明**：PicBerth 的内嵌前端管理界面当前仅提供简体中文。双语文档面向服务端部署与运维参考，不代表界面多语言支持。

- **公开访问与稳定链接**：所有已上传图片均为公开访问。图片采用随机 ID，移动和重命名不会改变链接；随机 ID 不提供访问控制。
- **最多 3 级文件夹**：支持文件夹层级归纳。移动图片或移动/重命名文件夹**不会改变**图片的公网访问链接。删除文件夹时要求其内无活跃图片与子文件夹。
- **开箱即用链接格式**：上传后可一键复制标准 URL、Markdown、HTML 与 BBCode 四种格式。
- **权限隔离的 API 令牌**：支持「上传」和「管理」两种权限范围。「上传」令牌仅能上传（可指定已有文件夹）并查看自己上传的图片；「管理」令牌可查看全部图片、管理文件夹、移动或删除图片。任何令牌均不能管理其他令牌或修改系统设置。
- **默认安全基线**：默认仅监听本地回环地址 `127.0.0.1:12433`；图片格式由内容判定，不信任扩展名，拒绝 SVG。网页初始化要求服务器日志中的一次性代码，也可通过离线 CLI 创建管理员；内置 CSP、速率限制与 Argon2id 密码哈希。
- **备份与校验**：内置一致性快照与数据校验，检查发现缺失或损坏文件时会标记相关图片；数据可通过快照在支持的平台之间迁移。

设计约束见 [`docs/product/constraints.md`](docs/product/constraints.md)；实现与接口说明见 [`docs/architecture.md`](docs/architecture.md)、[`docs/api.md`](docs/api.md)。

> **版本提示**：`v0.0.2` 正在准备，尚未发布。GitHub Releases 当前提供 `v0.0.1`；本仓库中的源码与 Compose 构建示例面向 v0.0.2 候选版本。

---

## 快速开始（原生二进制）

从 [GitHub Releases](https://github.com/dejavumoe/picberth/releases) 下载当前已发布版本的平台压缩包（提供 Linux AMD64 / ARM64、Windows AMD64、macOS ARM64），解压后即可运行，无需安装 Node.js 或其他运行时：

```sh
picberth serve --data-dir /var/lib/picberth --public-url https://pics.example.com
```

### 首次初始化

1. **通过网页初始化**：首次启动且数据库中尚无管理员时，服务器日志会在 `WARN` 级别打印一行一次性初始化代码（格式形如 `ABCD-EFGH-JKMN`）。使用浏览器打开公开地址进入 `/setup` 页面，输入该代码并设置管理员账号密码。
2. **离线命令行初始化**：也可以在启动服务前通过命令行直接创建管理员：

```sh
echo 'your-secure-password' | picberth admin create --username admin --password-stdin --data-dir /var/lib/picberth
```

随后在反向代理（Caddy 或 Nginx）中将公网域名反向代理至 `127.0.0.1:12433`。配置示例见 [`deploy/`](deploy/)。

---

## Docker 部署

当前源码构建的候选镜像使用 `gcr.io/distroless/static-debian13:nonroot`，以 UID/GID `65532:65532` 运行，不含 shell。Compose 示例启用只读根文件系统，并从当前检出构建镜像：

```sh
docker compose -f deploy/compose.yaml up -d --build
```

### 部署注意事项

- **切勿直接暴露端口**：示例中仅将端口绑定到宿主机的 `127.0.0.1:12433:12433`。**不要**配置为 `12433:12433`——Docker 发布端口会自动配置宿主机 iptables 规则并绕过 ufw / firewalld 防火墙，导致服务未经反代直接暴露于公网。
- **存储卷建议**：Docker Desktop（macOS / Windows）用户请务必使用 Docker 命名卷（如 `picberth-data:/data`），避免宿主机文件夹绑定挂载导致的性能损耗与文件所有权权限问题。
- **Docker 版本要求**：建议使用 Docker Engine ≥ 28.3.3（修复了 firewalld 重载后已发布端口暴露至局域网的安全漏洞 CVE-2025-54388，以及旧版回环端口绑定的隔离边界问题）。
- **内置 Caddy 示例**：若宿主机未安装独立反向代理，可使用 [`deploy/compose.caddy.yaml`](deploy/compose.caddy.yaml)，由同网络内的 Caddy 容器自动申请 TLS 证书并提供 HTTPS 接入。

---

## 反向代理与 TLS

PicBerth 设计为工作在反向代理之后，由代理层负责 TLS 终结与公网防护。

- **上传大小限制**：
  - **Nginx**：仓库示例配置 `client_max_body_size 110m;`。
  - **Caddy**：仓库示例在 `request_body` 块内配置 `max_size 110MB`。代理限额应高于 PicBerth 的上传硬上限，并留出 multipart 开销。
- **真实客户端 IP**：必须将反向代理的 IP 或 CIDR 填入 `PICBERTH_TRUSTED_PROXIES`（本机代理填 `127.0.0.1`；Docker 宿主机代理填 bridge 网关如 `172.30.112.1`）。PicBerth 从右向左解析 `X-Forwarded-For`，自动剥离受信任代理条目并忽略客户端伪造的头部。若未配置受信任代理，速率限制将把所有访客合并统计，且系统状态页会显示 `untrusted-forwarded` 告警。
- **公开地址与图片地址**：图片链接由 `PICBERTH_PUBLIC_URL` / `PICBERTH_IMAGE_URL` 生成，不使用请求的 Host 头部。更换域名后，已有链接仍指向旧地址；如需保留它们，应继续提供旧地址或设置重定向。

参考配置见 [`deploy/Caddyfile`](deploy/Caddyfile) 与 [`deploy/nginx.conf`](deploy/nginx.conf)。

---

## 配置

配置项按如下优先级覆盖生效：**命令行参数 > 环境变量 > 配置文件（`--config`，每行 `KEY=value`）> 默认值**。

| 环境变量 | 命令行参数 | 默认值 | 说明 |
|---|---|---|---|
| `PICBERTH_LISTEN` | `--listen` | `127.0.0.1:12433` | 监听地址与端口；绑定至非回环地址时会记录告警 |
| `PICBERTH_DATA_DIR` | `--data-dir` | 操作系统应用数据目录 | 存储目录；必须为本地文件系统，不支持 NFS / SMB 网络挂载 |
| `PICBERTH_PUBLIC_URL` | `--public-url` | `http://<listen>` | 管理界面的外部访问基准地址，也是默认图片外链基础地址 |
| `PICBERTH_IMAGE_URL` | `--image-url` | 同 `PUBLIC_URL` | 图片公开链接的基础地址，可配置为独立的图片域名或 CDN 地址 |
| `PICBERTH_TRUSTED_PROXIES` | `--trusted-proxies` | 空 | 受信任反向代理 IP 或 CIDR（逗号分隔）；本机代理填写 `127.0.0.1` |
| `PICBERTH_HARD_MAX_UPLOAD` | `--hard-max-upload` | `104857600` (100 MiB) | 单文件上传大小的硬上限（字节） |
| `PICBERTH_SNAPSHOT_DIR` | `--snapshot-dir` | `<数据目录>/snapshots` | 快照默认输出目录 |
| `PICBERTH_LOG_LEVEL` | `--log-level` | `info` | 日志级别（`debug`、`info`、`warn`、`error`） |
| `PICBERTH_LOG_FORMAT` | `--log-format` | `text` | 日志输出格式（`text` 或 `json`） |

> **覆盖规则**：显式设置为空的环境变量或参数会覆盖配置文件中的同名项。例如在命令行传入 `--trusted-proxies ''` 会清空配置文件中已有的受信任代理列表。

单次上传大小限制、允许的图片格式（JPEG、PNG、GIF、WebP、AVIF）、同目录文件去重、回收站保留天数与操作日志保留天数可在管理员登录后的「设置」页面在线调整。

---

## 接入上传工具

在管理后台「令牌」页面创建 API 令牌。创建成功弹窗提供开箱即用的配置片段，包括针对 curl、PicGo / PicList 和 ShareX 的专用规则。

### HTTP API 上传

```sh
curl -H "Authorization: Bearer pb_..." \
  -F file=@screenshot.png \
  -F folder=3n7vq2k9xwbd4f6h \
  https://pics.example.com/api/v1/images
```

- **参数说明**：文件内容通过 `file` 字段提交；可选字段 `folder` 为目标文件夹的 16 位 ID（留空上传至根目录）。PicBerth 不会自动创建不存在的文件夹，指定未知文件夹会返回 404。

### 第三方客户端配置

- **PicGo / PicList**（使用自带的自定义 Web 图床 / `web-uploader` 插件）：
  - API 地址：`https://pics.example.com/api/v1/images`
  - POST 参数名：`file`
  - JSON 路径：`url`
  - 自定义请求头：`{"Authorization": "Bearer pb_..."}`
- **ShareX**：在令牌创建对话框中下载生成的 `.sxcu` 配置文件，双击即可直接导入 ShareX 自定义上传目标。

### 权限范围说明

令牌权限范围严格分为两类：
- `upload`（仅上传）：允许上传文件（可指定已有文件夹），且仅能读取和检索由该令牌上传的图片。
- `manage`（管理）：允许检索与管理全部图片、管理文件夹结构、移动/重命名/删除/恢复图片。

所有 API 令牌均无权管理其他令牌或修改服务设置。接口规范可在管理后台直接浏览「令牌 → API 文档」（`/docs/api`），机器可读定义见 `/api/v1/openapi.json`。

---

## 备份、恢复与维护

### 创建快照与恢复

```sh
# 服务停止时通过 CLI 创建快照（服务运行中建议在「状态」页面在线创建）
picberth snapshot --out /backup/picberth

# 将 --from 指向上一条命令输出的时间戳子目录，其中包含 MANIFEST.json
picberth restore --from /backup/picberth/20261117T120000Z --data-dir /var/lib/picberth-new

# 校验文件哈希与结构完整性
picberth check --full --data-dir /var/lib/picberth-new
```

- **严禁直接复制运行中的数据库**：在线运行时直接复制 `picberth.db` 可能产生损坏的 WAL 状态。请务必使用内置快照命令（或状态页快照功能）生成一致性备份，再通过 restic、borg、rsync 等工具归档快照目录。
- **快照包含完整凭据**：快照包含数据库与全部图片引用，因此会完整恢复管理员账户凭据、活跃会话和 API 令牌。**只能从可信且防篡改的安全备份源执行恢复**。
- **文件权限与目录安全**：数据目录和备份目录严禁赋予其他非特权操作系统账户写入权限；PicBerth 数据库由 SQLite 通过绝对路径打开，并非独立文件系统沙箱。
- **敏感数据删除注意**：历史快照和自动迁移备份拥有独立保留周期，不会随线上会话注销、令牌撤销或日志到期而自动清理。清理历史敏感数据时须同步处理历史备份。
- **自动迁移与回滚**：程序启动时会自动执行 SQLite 迁移。在执行架构迁移前，系统会自动将元数据库副本备份至 `<数据目录>/backups/picberth-schema<N>-<UTC>.db`（最多保留最新的 5 份）。若数据库架构版本高于程序支持版本，程序将拒绝启动；此时切回旧版二进制并将该备份替换回 `picberth.db` 即可安全回滚。
- **完整性检查机制**：
  - **快速检查**（`quick`）：校验活跃与回收站记录的文件路径与大小，自动补全中断的移动操作，清理过期上传临时碎片。
  - **完整检查**（`full`）：在快速检查基础上重新计算并比对所有文件的 SHA-256 哈希，将无引用的孤立文件移入 `<数据目录>/quarantine/`（绝不会静默删除）。只有通过完整检查才能清除图片的损坏标记（`broken_at`）。

---

## 开发

本项目使用 [mise](https://mise.jdx.dev) 统一管理和锁定本地开发工具链版本：

```sh
mise install              # 安装 mise.toml 中锁定的 Go、Node 与 pnpm
mise run web:install      # 安装前端依赖（使用 frozen-lockfile）
mise run dev              # 启动 Go 服务（数据目录位于 .devdata）
mise run web:dev          # 另开终端：启动 Vite 开发服务器（http://127.0.0.1:5173，自动代理 /api 与 /i）
mise run test             # 执行完整测试套件（go vet + go test + svelte-check + E2E 类型检查）
mise run web:e2e:install  # 安装 Playwright Chromium 及系统依赖
mise run web:e2e          # 执行 Playwright 端到端集成测试（依赖构建产物，使用独立测试目录）
mise run build            # 构建最终发行单二进制文件（生成 dist/picberth，已内嵌前端 SPA）
```

> **注意**：`test`、`dev` 和 `build` 任务会自动先编译前端内嵌资源。在干净的新检出工作区中，若跳过 mise 任务直接执行原生 `go test ./...`、`go vet ./...` 或 `go run ./cmd/picberth`，请务必先运行一次 `mise run web:build`。
>
> 在 Windows 开发环境下，建议将浏览器依赖安装及 Playwright E2E 测试放入 WSL 环境中执行以保证与 CI 环境一致。

```powershell
linux-task.ps1 -Mode build -Project <项目路径> -Command 'export PATH="$HOME/.local/bin:$PATH"; mise run web:e2e:install && mise run web:e2e'
```

---

## 发布与校验

发布演练与正式发布流程见 [`docs/releasing.md`](docs/releasing.md)。

压缩包包含程序与 MIT 许可证，`SHA256SUMS` 是单独的 Release 附件；压缩包同时有 GitHub 构建证明。发布的容器镜像包含 SBOM 与构建来源证明。

项目采用 [MIT 许可证](LICENSE)。
