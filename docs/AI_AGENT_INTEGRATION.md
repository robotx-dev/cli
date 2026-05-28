# AI Agent Integration

本指南描述如何把 RobotX CLI 集成到 Agent / CI 系统中。

## 面向公开页面的推荐心智

如果你的 Agent / Claw 需要对外持续发布结果，优先把 RobotX 视为“发布层 / 结果站点层”，而不是单纯部署工具：

- Runtime 负责生产结果
- RobotX 负责发布身份、历史、成果与复制入口

面向这类用法的具体工作流见 `skills/agent-pages/SKILL.md`，底层部署细节见 `skills/robotx/SKILL.md`。

## 推荐集成路径

1. 使用 release 二进制安装（不要依赖本地 Go）
2. 使用 `--output json` 获取稳定契约
3. 用 `robotx doctor --output json` 做凭证和连通性预检
4. 通过 shell skill / GitHub Action 调用 CLI
5. 暂不使用 MCP（当前未实现）

## 1) 安装 CLI（二进制）

```bash
curl -fsSL https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh | bash
```

大陆网络或 GitHub 下载慢时：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.6 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash
```

## 2) 登录或传入凭证

交互式环境优先使用登录：

```bash
robotx login --base-url https://robotx.xin
robotx doctor --output json
```

CI 和非交互环境使用环境变量：

```bash
export ROBOTX_BASE_URL=https://robotx.xin
export ROBOTX_API_KEY=your-api-key
```

## 3) 用 JSON 模式调用

```bash
robotx deploy . --create --target main --name my-app --output json
robotx deploy . --update --target main --output json
robotx targets --output json
robotx targets remove main --output json
robotx access status --project-id proj_123 --output json
robotx access open --project-id proj_123 --output json
robotx projects --limit 50 --output json
robotx versions --project-id proj_123 --output json
robotx status --project-id proj_123 --output json
robotx publish --project-id proj_123 --build-id build_456 --output json
robotx projects delete --project-id proj_123 --yes --output json
```

说明：

- stdout 只输出 JSON
- stderr 输出进度日志

### JSON 成功响应

```json
{
  "success": true,
  "command": "status",
  "data": {
    "project": {
      "project_id": "proj_123"
    }
  }
}
```

### JSON 失败响应（stderr）

```json
{
  "success": false,
  "error": {
    "code": "api_error",
    "message": "failed to get project"
  }
}
```

## 4) Python 调用示例

```python
import json
import subprocess

cmd = [
    "robotx", "deploy", ".",
    "--create",
    "--target", "main",
    "--name", "my-app",
    "--output", "json",
]
result = subprocess.run(cmd, capture_output=True, text=True)

if result.returncode == 0:
    payload = json.loads(result.stdout)
    print(payload["data"].get("project_id"))
else:
    err_line = result.stderr.strip().splitlines()[-1]
    err = json.loads(err_line)
    raise RuntimeError(err["error"]["message"])
```

## 5) GitHub Action

仓库内置 `action.yml`，可直接使用：

```yaml
- uses: robotx-dev/cli@v0.6
  with:
    base-url: ${{ secrets.ROBOTX_BASE_URL }}
    api-key: ${{ secrets.ROBOTX_API_KEY }}
    project-path: .
    project-name: my-app
    access: open
    verify-url: "true"
    # 可选：使用 action 源码构建 CLI（而非 release 二进制）
    # version: source
    # 推荐：CI 固定更新已有项目时，透传 project id
    # extra-args: --project-id ${{ secrets.ROBOTX_PROJECT_ID }}
```

如果 CI 需要按项目名复用已有项目，可显式使用 `extra-args: --upsert`。不要依赖临时 checkout 里的 `.robotx/targets.json` 持久化目标记录。

Action 输出：

- `project_id`
- `build_id`
- `status`
- `preview_url`
- `production_url`
- `version_label`
- `version_seq`
- `source_ref`
- `raw_json`

## 6) 命令约束

- `deploy`：新建项目建议用 `--create --target <name> --name <project>`；更新已有目标建议用 `--update --target <name>`
- `targets remove`：只删除本地 `.robotx/targets.json` 记录，不删除远端项目
- `status`：`--project-id` 与 `--build-id` 至少提供一个
- `versions`：必须带 `--project-id`
- `projects delete`：删除远端项目，必须先获得用户明确确认，并传 `--yes`
- `status --logs` 与 `logs`：不再支持，因为 RobotX 不再提供远程 build 日志

## MCP 说明

`robotx mcp` 当前是占位实现，不应作为生产集成方案。Agent 集成请使用 shell/CLI 模式。
