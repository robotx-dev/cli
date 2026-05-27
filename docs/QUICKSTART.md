# RobotX CLI 快速开始

## 1) 安装（推荐二进制）

```bash
curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | bash
robotx --version
```

大陆网络或 GitHub release 下载很慢时，默认推荐使用 `mr.robotx.xin` 中转，并固定版本避免卡在 `latest` 解析：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.3 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash
robotx --version
```

或使用 Go（自动写入 PATH）：

```bash
curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/go-install.sh | bash
robotx --version
```

> 脚本会优先安装 `cmd/robotx@latest`，并在旧版本仓库结构下自动回退并创建 `robotx` 命令别名。

若只想用纯 `go install`：

```bash
go install github.com/haibingtown/robotx_cli/cmd/robotx@latest
```

## 2) 配置

先登录自动写入（推荐）：

```bash
robotx login --base-url https://your-robotx-server.com
```

或手动配置：

```bash
export ROBOTX_BASE_URL=https://your-robotx-server.com
export ROBOTX_API_KEY=your-api-key
```

或写入 `~/.robotx.yaml`：

```yaml
base_url: https://your-robotx-server.com
api_key: your-api-key
```

## 3) 部署

```bash
robotx deploy . --name my-app --output json
```

默认会使用 `--local-build=true` 并在成功后 `--publish=true`；如只想预览可追加 `--publish=false`。
RobotX 不再支持云端 build，`--local-build` 必须保持为 `true`。
如需对齐 CI 标识，可追加 `--version-label v1.2.3 --source-ref "tag:v1.2.3@<sha>"`。
`--name` 需符合服务端规则：`4-63` 位，仅允许小写字母/数字/`-`。

## 4) 查询状态

```bash
robotx status --project-id proj_123 --output json
robotx status --build-id build_456 --output json
```

## 5) 发布

```bash
robotx publish --project-id proj_123 --build-id build_456 --output json
```

## 6) 常见参数

- `--output json` / `--json`: 机器可读输出
- `--publish`: 构建成功后自动发布（默认 `true`，可用 `--publish=false` 关闭）
- `--local-build`: 本地构建并上传产物（默认 `true`；RobotX 不再支持 `false`）
- `--wait=false`: 不等待构建结束
- `--timeout 900`: 自定义等待超时
- `--version-label`: 自定义版本号（不传由服务端自动递增）
- `--source-ref`: 来源标识（如 tag/branch/commit）

## 注意

- `robotx mcp` 当前未实现（占位功能）
- JSON 模式下 stdout 仅输出 JSON，进度日志写入 stderr
