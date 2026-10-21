# PicBerth

面向个人与小团队的自托管图床：**一个可执行文件 + 一个数据目录**。图片保存在本地磁盘，SQLite 只保存元数据；没有外部数据库、缓存或对象存储依赖。

- 上传：网页拖拽 / 粘贴 / 多选，逐个显示进度；HTTP API 兼容 PicGo、PicList、ShareX 与脚本。
- 链接长期稳定：随机且不可枚举的 ID，链接对应的字节永不改变，删除后 ID 也不会被复用。
- 管理：图库检索与筛选、批量操作、回收站（默认保留 30 天）、带权限范围的 API 令牌。
- 默认安全：只监听 `127.0.0.1:12433`；类型由内容判定（不信任扩展名）；拒绝 SVG；首个管理员只能用服务器日志中的一次性初始化代码创建。
- 可恢复：内置一致性快照与恢复校验，数据目录可在 Linux、Windows、macOS 与 Docker 之间迁移。

设计依据见 [`PicBerth-设计研究基线-v0.1.md`](PicBerth-设计研究基线-v0.1.md)；实现说明见 [`docs/architecture.md`](docs/architecture.md)、[`docs/api.md`](docs/api.md)、[`docs/ui.md`](docs/ui.md)。

## 快速开始（原生二进制）

```sh
picberth serve --data-dir /var/lib/picberth --public-url https://pics.example.com
```

首次启动时日志会打印一行初始化代码（形如 `ABCD-EFGH-JKMN`）。用浏览器打开公开地址，在初始化页面输入该代码并创建管理员。也可以离线创建：

```sh
echo 'a-long-password' | picberth admin create --username admin --password-stdin
```

然后在反向代理（Nginx 或 Caddy）中把域名转发到 `127.0.0.1:12433`，示例见 [`deploy/`](deploy/)。

## Docker

```sh
docker compose -f deploy/compose.yaml up -d
```

- 示例只把端口发布到宿主机的 `127.0.0.1`。**不要**写成 `12433:12433`：Docker 发布的端口会绕过 ufw / firewalld，直接暴露在公网。
- 需要 Docker Engine ≥ 28.3.3。Docker Desktop（macOS / Windows）请使用命名卷，不要把数据目录绑定挂载到宿主机文件夹。
- 不想在宿主机安装代理时，可以用 [`deploy/compose.caddy.yaml`](deploy/compose.caddy.yaml)，由同一网络中的 Caddy 负责 HTTPS。

## 配置

优先级：命令行参数 > 环境变量 > 配置文件（`--config`，每行 `KEY=value`）> 默认值。

| 环境变量 | 参数 | 默认值 | 说明 |
|---|---|---|---|
| `PICBERTH_LISTEN` | `--listen` | `127.0.0.1:12433` | 监听地址；改为非回环地址时会告警 |
| `PICBERTH_DATA_DIR` | `--data-dir` | 平台惯例目录 | 必须位于本地磁盘，不能放在 NFS / SMB 上 |
| `PICBERTH_PUBLIC_URL` | `--public-url` | `http://<listen>` | 访问管理界面的地址，也是生成链接的依据 |
| `PICBERTH_IMAGE_URL` | `--image-url` | 同公开地址 | 图片链接的基础地址，推荐使用独立域名 |
| `PICBERTH_TRUSTED_PROXIES` | `--trusted-proxies` | 空 | 反向代理的 IP / CIDR，逗号分隔；本机代理填 `127.0.0.1` |
| `PICBERTH_HARD_MAX_UPLOAD` | `--hard-max-upload` | `104857600` | 单文件大小的硬上限（字节） |
| `PICBERTH_SNAPSHOT_DIR` | `--snapshot-dir` | `<数据目录>/snapshots` | 快照输出目录 |
| `PICBERTH_LOG_LEVEL` / `PICBERTH_LOG_FORMAT` | `--log-level` / `--log-format` | `info` / `text` | 日志级别 / 格式（`text` 或 `json`） |

上传限制、允许的格式、回收站保留天数、会话时长等业务设置在管理界面的「设置」中修改。

### 反向代理要点

- **请求体上限**：Nginx 默认只允许 1 MB，必须调大 `client_max_body_size`；Caddy 建议显式设置 `request_body max_size`。
- **真实 IP**：把代理地址写入 `PICBERTH_TRUSTED_PROXIES`。PicBerth 从右向左解析 `X-Forwarded-For`，客户端伪造的条目会被忽略。如果没有配置，限流会把所有访问者当成同一个地址，状态页也会给出告警。
- **链接**：链接只由 `PICBERTH_PUBLIC_URL` / `PICBERTH_IMAGE_URL` 生成，不读取请求头。修改它们会使已经发出的链接失效。

## 接入上传工具

在「令牌」页面新建令牌，创建完成的对话框里有可直接复制的 curl、PicGo 与 ShareX 配置。

```sh
curl -H "Authorization: Bearer pb_…" -F file=@shot.png https://pics.example.com/api/v1/images
```

- **PicGo / PicList**（web-uploader 插件）：API 地址 `https://pics.example.com/api/v1/images`，POST 参数名 `file`，JSON 路径 `url`，自定义请求头 `{"Authorization":"Bearer pb_…"}`。
- **ShareX**：在令牌对话框中下载 `.sxcu` 文件后双击导入。

令牌分为三种权限：「上传」只能上传并查看自己上传的图片，「只读」可查看全部图片，「管理」还可以修改可见性和删除图片。任何令牌都不能管理令牌或修改设置。

## 备份与恢复

```sh
picberth snapshot --out /backup/picberth-2026-10-21     # 服务停止时使用；运行中请在「状态」页面创建快照
picberth restore --from /backup/picberth-2026-10-21 --data-dir /var/lib/picberth-new
picberth check --full                                    # 校验每个文件的哈希并找出孤儿文件
```

- **不要**直接复制正在运行的 `picberth.db`，这可能得到损坏的副本。请先生成快照，再用 restic、borg、rsync 等工具备份快照目录。
- 快照包括数据库，以及数据库引用的全部图片（回收站中的图片也包括在内）。复制过程中会逐个校验哈希，任何不一致都会让快照失败。
- 升级时会自动执行数据库迁移。迁移前会把元数据库复制到 `<数据目录>/backups/`（图片文件在迁移中不会改动）。如果数据库版本比程序新，程序会拒绝启动；这时换回旧版本程序，并用这份备份替换 `picberth.db` 即可回滚。

## 开发

需要 [mise](https://mise.jdx.dev)。

```sh
mise install            # Go、Node、pnpm 的锁定版本
mise run web:install
mise run dev            # Go 服务，数据目录为 .devdata
mise run web:dev        # 另开一个终端：Vite 开发服务器 http://127.0.0.1:5173
mise run test           # go vet + go test + svelte-check
mise run build          # 生成 dist/picberth（内嵌前端）
```
