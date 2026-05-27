# RobotX CLI

RobotX CLI 用于将应用部署到 RobotX 平台，支持 `login` / `deploy` / `access` / `doctor` / `projects` / `versions` / `status` / `publish`。

## 当前状态

- CLI 集成（shell/CI/Agent）: 可用
- JSON 机器输出: 可用（`--output json` 或 `--json`）
- MCP 模式（`robotx mcp`）: 未实现（占位）

## 文档导航

- 文档索引: [docs/README.md](docs/README.md)
- 快速开始: [docs/QUICKSTART.md](docs/QUICKSTART.md)
- 示例集合: [docs/EXAMPLES.md](docs/EXAMPLES.md)
- Agent 集成: [docs/AI_AGENT_INTEGRATION.md](docs/AI_AGENT_INTEGRATION.md)
- Skills 总览: [skills/README.md](skills/README.md)
- 项目文档归档: `docs/`

## 安装

### 方式 1: 下载安装脚本（推荐，无需 Go）

```bash
curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | bash
```

可选参数：

- `ROBOTX_VERSION=latest`（默认）或 `vX.Y.Z`
- `ROBOTX_INSTALL_DIR=$HOME/.local/bin`
- `ROBOTX_REPO=haibingtown/robotx_cli`
- `ROBOTX_AUTO_PATH=1`（默认，自动写入 shell profile）或 `0`
- `ROBOTX_CONNECT_TIMEOUT=10`（默认连接超时秒数）
- `ROBOTX_MAX_TIME=300`（默认单次下载最长秒数）
- `ROBOTX_DOWNLOAD_RETRIES=3`（默认下载重试次数）
- `ROBOTX_RETRY_DELAY=2`（默认重试间隔秒数）
- `ROBOTX_CURL_PROGRESS=1`（本地默认显示下载进度；CI 默认关闭）
- `ROBOTX_GITHUB_API_BASE=https://api.github.com`（解析 latest 使用）
- `ROBOTX_GITHUB_PROXY=https://your-proxy.example.com`（给 GitHub release URL 加代理前缀）
- `ROBOTX_DOWNLOAD_BASE_URL=https://your-mirror.example.com/haibingtown/robotx_cli/releases/download`（直接使用 release 资产镜像）

大陆网络默认推荐使用 `mr.robotx.xin` 中转，并固定版本跳过 GitHub API：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.3 \
      ROBOTX_GITHUB_PROXY=https://mr.robotx.xin \
      bash
```

如果你有内部 release 镜像，优先使用镜像源：

```bash
curl -fsSL https://your-mirror.example.com/haibingtown/robotx_cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.3 \
      ROBOTX_DOWNLOAD_BASE_URL=https://your-mirror.example.com/haibingtown/robotx_cli/releases/download \
      bash
```

### 方式 2: 从源码安装

```bash
go install github.com/haibingtown/robotx_cli/cmd/robotx@latest
```

### 方式 3: 使用 Go 安装并自动配置 PATH

```bash
curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/go-install.sh | bash
```

可选参数：

- `ROBOTX_GO_PACKAGE=github.com/haibingtown/robotx_cli/cmd/robotx@latest`
- `ROBOTX_LEGACY_GO_PACKAGE=github.com/haibingtown/robotx_cli@latest`（主包安装失败时回退）
- `ROBOTX_INSTALL_DIR=$HOME/.local/bin`
- `ROBOTX_AUTO_PATH=1`（默认，自动写入 shell profile）或 `0`

说明：纯 `go install ...` 命令本身不会自动修改你的 shell 环境变量（PATH），这是 Go 工具链行为；如需“安装后直接可用”建议用方式 1 或方式 3。

## 配置

支持配置文件 `~/.robotx.yaml`：

```yaml
base_url: https://robotx.xin
api_key: your-api-key
```

或使用环境变量：

```bash
export ROBOTX_BASE_URL=https://robotx.xin
export ROBOTX_API_KEY=your-api-key
```

也可使用 Web 登录自动写入凭证：

```bash
robotx login --base-url https://robotx.xin
```

## 输出模式

- `--output text`（默认）: 面向人类阅读
- `--output json` 或 `--json`: 面向程序解析

在 JSON 模式下：

- stdout: 仅 JSON 结果
- stderr: 进度日志/诊断信息

成功输出结构：

```json
{
  "success": true,
  "command": "deploy",
  "data": {
    "project_id": "proj_xxx",
    "build_id": "build_xxx"
  }
}
```

失败输出结构（stderr 最后一行）：

```json
{
  "success": false,
  "error": {
    "code": "api_error",
    "message": "failed to resolve project"
  }
}
```

## 命令

### deploy

部署新项目或更新当前工作区已记录的项目。默认新建项目时会避免覆盖同名旧项目；如需旧版“按名称复用”行为，显式使用 `--upsert --name my-app`。

```bash
robotx deploy [project-path] \
  [--name my-app] \
  [--version-label v1.2.3] [--source-ref "tag:v1.2.3@<sha>"] \
  [--publish=true] [--local-build=true] [--wait=true] [--timeout 600] \
  [--access unchanged|open|login|private] [--verify-url]
```

项目名规则（与服务端一致）：长度 4-63，仅允许小写字母/数字/`-`，且首尾必须是字母或数字。

默认行为：

- `--local-build=true`：本地构建并上传产物
- `--publish=true`：构建成功后自动发布
- `--version-label`：显式指定部署版本号（不传则服务端按数字递增）
- `--source-ref`：记录来源标识（建议在 CI 中传 `tag/branch + commit`）
- `--access`：部署成功后更新访问策略；默认 `unchanged`
- `--verify-url`：显式检查生产链接是否能匿名打开
- Preview 链接默认仅项目 owner 可访问；生产访问策略以 publish 版本策略为准
- `--visibility public` 只是项目可见性，不等于“未登录可直接访问”；匿名公开请用 `--access open` 或 `robotx access open`
- RobotX 不再支持云端 build；`--local-build` 只能保持为 `true`

本地构建模式（默认开启）：

```bash
robotx deploy . --name my-app --local-build \
  [--install-command "npm ci"] \
  [--build-command "npm run build"] \
  [--output-dir dist]
```

### login

通过设备码 + 浏览器授权登录，并自动写入 API 凭证到配置文件：

```bash
robotx login --base-url https://robotx.xin
```

常用参数：

- `--device-start-path`：设备登录启动接口（默认 `/api/auth/device/start`）
- `--device-poll-path`：设备登录轮询接口（默认 `/api/auth/device/poll`）
- `--timeout`：登录超时秒数（默认 `180`）
- `--no-browser`：不自动打开浏览器，仅打印登录链接

### projects

查询当前账号下的项目列表：

```bash
robotx projects [--limit 50]
```

### access

查看或修改项目访问策略：

```bash
robotx access status --project-id proj_123
robotx access open --project-id proj_123      # 未登录可直接访问
robotx access login --project-id proj_123     # 需要 RobotX 登录
robotx access private --project-id proj_123   # 白名单私有
```

说明：`open` 会设置匿名公开访问；`login` 是公开项目但需要平台登录；`private` 是白名单访问。

### doctor

检查本机 CLI、配置和登录态，不会创建或修改项目：

```bash
robotx doctor
robotx doctor --output json
```

### versions

查看项目最近构建版本（用于多版本管理和回滚前选择）：

```bash
robotx versions --project-id proj_123 [--limit 20]
```

`versions` 也支持别名：`robotx builds --project-id proj_123`。

### status

查询项目和/或构建状态：

```bash
robotx status [--project-id proj_123] [--build-id build_456]
```

说明：

- `--project-id` 与 `--build-id` 至少提供一个
- `status --logs` 和 `robotx logs` 已不再可用，因为 RobotX 不再提供远程 build 日志

### publish

发布构建到生产环境：

```bash
robotx publish --project-id proj_123 --build-id build_456
```

### mcp

```bash
robotx mcp
```

当前返回未实现错误（占位功能）。

## GitHub Action

仓库根目录提供了 composite action（[action.yml](action.yml)），默认流程是：

1. 下载 release 二进制
2. 校验 checksum
3. 执行 `robotx deploy --publish=true --output json`
4. 输出 `project_id/build_id/status/url/version_label/version_seq/source_ref` 等字段

示例工作流见：`.github/workflows/action-example.yml`。

补充：

- 支持输入别名：`base_url`/`api_key`（等价于 `base-url`/`api-key`）
- 支持输入别名：`version_label`/`source_ref`（等价于 `version-label`/`source-ref`）
- 未显式传 `source-ref` 时，action 会默认使用 `GITHUB_REF` + `GITHUB_SHA` 生成来源标识
- 支持 `access: open|login|private|unchanged` 和 `verify-url: true`
- `version: source` 可在 CI 中直接从 action 源码构建 CLI（适合验证 `@main` 最新变更）

## Release

标签推送触发自动发布：

- Workflow: `.github/workflows/release.yml`
- 产物：
  - `robotx_<version>_<os>_<arch>.tar.gz`（linux/darwin）
  - `robotx_<version>_<os>_<arch>.zip`（windows）
  - `checksums.txt`

## 退出码

- `1`: 参数/配置/通用错误
- `2`: API/网络错误
- `3`: 构建失败
- `4`: 发布失败
