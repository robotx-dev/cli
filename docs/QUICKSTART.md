# RobotX CLI 快速开始

## 1) 安装（推荐二进制）

```bash
curl -fsSL https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh | bash
robotx --version
```

大陆网络或 GitHub release 下载很慢时，默认推荐使用 `mr.robotx.xin` 中转，并固定版本避免卡在 `latest` 解析：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.6 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash
robotx --version
```

如果 `mr.robotx.xin` 下载 GitHub release 包时返回 502 或超时，使用备用线路下载 release 包：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.6 ROBOTX_GITHUB_PROXY=https://gh-proxy.com bash
robotx --version
```

或使用 Go（自动写入 PATH）：

```bash
curl -fsSL https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/go-install.sh | bash
robotx --version
```

> 脚本会优先安装 `cmd/robotx@latest`，并在旧版本仓库结构下自动回退并创建 `robotx` 命令别名。

若只想用纯 `go install`：

```bash
go install github.com/robotx-dev/cli/cmd/robotx@latest
```

## 2) 登录或配置

先登录自动写入（推荐）：

```bash
robotx login --base-url https://robotx.xin
robotx doctor --output json
```

CI 或非交互环境手动配置：

```bash
export ROBOTX_BASE_URL=https://robotx.xin
export ROBOTX_API_KEY=your-api-key
```

或写入 `~/.robotx.yaml`：

```yaml
base_url: https://robotx.xin
api_key: your-api-key
```

## 3) 部署

```bash
robotx deploy . --create --target main --name my-app --output json
```

默认会使用 `--local-build=true` 并在成功后 `--publish=true`；如只想预览可追加 `--publish=false`。
RobotX 不再支持云端 build，`--local-build` 必须保持为 `true`。
如需对齐 CI 标识，可追加 `--version-label v1.2.3 --source-ref "tag:v1.2.3@<sha>"`。
`--name` 需符合服务端规则：`4-63` 位，仅允许小写字母/数字/`-`。
如需生产链接匿名可访问，部署时追加 `--access open --verify-url`。

后续更新同一个目标：

```bash
robotx deploy . --update --target main --output json
```

## 4) 查询状态

```bash
robotx status --project-id proj_123 --output json
robotx status --build-id build_456 --output json
```

## 5) 发布

```bash
robotx publish --project-id proj_123 --build-id build_456 --output json
```

## 6) 访问、本地记录和删除

```bash
robotx access open --project-id proj_123 --output json
robotx targets --output json
robotx targets remove main --output json
robotx projects delete --project-id proj_123 --yes --output json
```

说明：

- `access open` 允许生产链接匿名访问
- `targets remove` 只删除本地 `.robotx/targets.json` 记录
- `projects delete` 删除远端项目，是破坏性操作，必须先确认用户意图

## 7) 常见参数

- `--output json` / `--json`: 机器可读输出
- `--create`: 创建新项目，同名项目存在时失败
- `--update`: 更新已有目标或指定项目
- `--target`: 使用 `.robotx/targets.json` 中的本地目标记录
- `--publish`: 构建成功后自动发布（默认 `true`，可用 `--publish=false` 关闭）
- `--local-build`: 本地构建并上传产物（默认 `true`；RobotX 不再支持 `false`）
- `--wait=false`: 不等待构建结束
- `--timeout 900`: 自定义等待超时
- `--version-label`: 自定义版本号（不传由服务端自动递增）
- `--source-ref`: 来源标识（如 tag/branch/commit）

## 注意

- `robotx mcp` 当前未实现（占位功能）
- `robotx logs` 和 `status --logs` 当前不可用
- JSON 模式下 stdout 仅输出 JSON，进度日志写入 stderr
