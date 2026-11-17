# 发布

当前准备版本为 `v0.0.2`。演练成功不代表已经发布。

## 发布演练

在 GitHub Actions 的 **Release → Run workflow** 中选择 `master`，版本填写 `v0.0.2`：

```sh
gh workflow run release.yml --ref master -f version=v0.0.2
```

演练执行完整 CI，构建四个平台的压缩包及双架构容器。四个 `binary-*` 与两个 `image-*` artifact 保留 7 天；后者包含原生 CPU 构建的 OCI 归档。不会创建标签、发布 Release、推送镜像或上传 GitHub 构建证明。
下载并检查四个压缩包内的程序、MIT 许可证和校验和，核对程序的版本、提交 SHA；遍历 OCI index，检查 amd64/arm64 平台、ELF 架构、运行用户和 SBOM/provenance 描述符。构建成功不能替代产物核验。
演练和正式发布都要求目标提交已包含在远程 `master` 中；非默认分支上的新提交不能直接进入发布流水线。

## 正式发布

1. 更新 `web/package.json`、`internal/buildinfo/buildinfo.go`、Dockerfile 和 OpenAPI 的版本，并维护 `docs/releases/<tag>.md`。开发构建保留 `-dev` 后缀。
2. 确认目标提交位于 `master`，该提交的 CI、CodeQL、Scorecard 和发布演练全部成功；检查未解决的安全告警。
3. 在 Windows 正式仓库内为这个已验证提交创建并推送标签：

   ```sh
   git tag -a v0.0.2 <verified-commit-sha> -m "PicBerth v0.0.2"
   git push origin v0.0.2
   ```

4. 标签触发正式发布。完成后下载附件，验证 `SHA256SUMS` 与 GitHub 构建证明，检查 `ghcr.io/dejavumoe/picberth:v0.0.2` 的 amd64 / arm64 平台、SBOM 和 provenance。使用独立临时数据目录进行启动与健康检查。

CI 的 `ci-ok` 继续作为默认分支门禁。普通分支通过 PR 验证，`maintenance/*` 可在快进默认分支前验证提交；Dependabot 不再重复触发 push CI。不要为了发版跳过门禁或覆写已有版本标签。
测试、交叉编译、E2E 与双架构 Docker 构建在变更检测后独立运行；`ci-ok` 仍汇总全部结果。新增并行不跳过失败或未运行的任务。

## 原生 CPU 构建

CI Go 测试与镜像分别在 `ubuntu-24.04`（AMD64）和 `ubuntu-24.04-arm`（ARM64）运行。
Release 附件使用与目标 CPU 相同的 Linux runner；Windows AMD64 和 macOS ARM64 仍是操作系统交叉编译，不代表在 Windows/macOS 上执行过测试。
镜像使用独立 BuildKit，不使用 Buildx 或 QEMU。普通 CI 和演练按架构缓存中间层，每次保存只保留本次构建可达的缓存；正式镜像不导入这些缓存。
发布附件不恢复 mise、pnpm、Go 模块和编译缓存，执行 `go mod verify` 并通过 `go build -a` 重新编译；普通 CI 保留缓存并验证 Go 模块。
两个架构先推送带 run ID、attempt 和架构的暂存标签，附件与镜像全部成功后才合并正式标签，保留 SBOM/provenance。
演练输出 OCI 归档到 runner 临时目录，不推送仓库。原生 runner 的实际排队时间和冷/热缓存耗时需通过 Actions 演练测量。

正式版本使用同一个完整标签：Git tag 和 GitHub Release 为 `v0.0.2`，Docker 镜像为 `ghcr.io/dejavumoe/picberth:v0.0.2`。不自动发布 `0.0.2`、`v0.0` 或 `latest` 别名。程序内嵌及 API 返回版本统一为 `v0.0.2`；npm 包管理元数据仍使用 `0.0.2`，源码开发默认值为 `v0.0.2-dev`。发布日志按中文、英文两节维护，功能、安装、安全提示和附件说明保持一致。
