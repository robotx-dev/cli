# RobotX CLI Examples

这些示例只覆盖当前支持的能力。Agent、CI 和脚本集成应优先使用 `--output json`，不要解析面向人的文本输出。

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh | bash
```

大陆网络推荐固定当前版本并使用 `mr.robotx.xin` 中转：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/robotx-dev/cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.6 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash
```

## 登录和检查

```bash
robotx login --base-url https://robotx.xin
robotx doctor --output json
```

CI 或非交互环境使用环境变量：

```bash
export ROBOTX_BASE_URL=https://robotx.xin
export ROBOTX_API_KEY=your-api-key
```

## 新建并发布

```bash
robotx deploy . \
  --create \
  --target main \
  --name my-app \
  --access open \
  --verify-url \
  --output json
```

说明：

- `--create` 会创建新项目；同名项目存在时失败。
- `--target main` 会把远端项目写入本地 `.robotx/targets.json`。
- `--access open` 才表示生产链接允许匿名访问；`--visibility public` 不等于匿名可访问。
- RobotX 当前只支持本地构建上传产物，`--local-build` 必须保持为 `true`。

## 更新已有目标

```bash
robotx deploy . \
  --update \
  --target main \
  --version-label v1.2.3 \
  --source-ref "tag:v1.2.3@$(git rev-parse HEAD)" \
  --output json
```

需要明确更新某个远端项目时：

```bash
robotx deploy . --update --project-id proj_123 --output json
```

只有在明确需要旧版“按项目名创建或复用”行为时，才使用：

```bash
robotx deploy . --upsert --name my-app --output json
```

## 本地目标记录

```bash
robotx targets --output json
robotx targets remove main --output json
```

`targets remove` 只删除本地记录，不删除远端 RobotX 项目。

## 访问策略

```bash
robotx access status --project-id proj_123 --output json
robotx access open --project-id proj_123 --output json
robotx access login --project-id proj_123 --output json
robotx access private --project-id proj_123 --output json
```

## 版本、状态和发布

```bash
robotx versions --project-id proj_123 --limit 20 --output json
robotx status --project-id proj_123 --build-id build_456 --output json
robotx publish --project-id proj_123 --build-id build_456 --output json
```

`versions` 也支持别名：

```bash
robotx builds --project-id proj_123 --output json
```

## 删除远端项目

远端项目删除是破坏性操作，必须先确认用户确实要删除项目，再执行：

```bash
robotx projects delete --project-id proj_123 --yes --output json
```

删除远端项目不会自动清理本地 `.robotx/targets.json`。需要时再删除本地目标记录：

```bash
robotx targets remove main --output json
```

## GitHub Actions

```yaml
- uses: robotx-dev/cli@v0.6
  with:
    base-url: ${{ secrets.ROBOTX_BASE_URL }}
    api-key: ${{ secrets.ROBOTX_API_KEY }}
    project-path: .
    project-name: my-app
    access: open
    verify-url: "true"
```

CI 固定更新已有项目时，用 `extra-args` 传入项目 ID：

```yaml
extra-args: --project-id ${{ secrets.ROBOTX_PROJECT_ID }}
```

如果 CI 需要按项目名复用已有项目，可显式使用 `extra-args: --upsert`。不要依赖临时 checkout 里的 `.robotx/targets.json` 持久化目标记录。

## 不再支持的能力

- `robotx logs` 和 `status --logs` 不再提供远程构建日志。
- `robotx mcp` 当前是占位命令，不作为生产集成方式。
