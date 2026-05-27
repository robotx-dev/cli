---
name: robotx-deploy-cn
description: 当用户或 Agent 需要把网站、应用、静态页面、Agent 公开页发布到 RobotX，或查询/发布 RobotX 项目版本时使用。特别适合无技术背景或技术背景较少的用户：Agent 应该用中文低门槛引导，少问问题，隐藏 CLI/API/JSON 细节，自动完成认证预检、部署、验证、错误翻译和结果汇报。不要用于开发 RobotX 服务端或前端、直接调用 runtime API、MCP 模式、远程构建日志或页面内容创作。
metadata:
  short-description: 面向低技术用户的 RobotX 中文发布技能
---

# RobotX 中文发布技能

用这个技能时，你的目标不是教用户理解部署，而是替用户把东西可靠发布出去，并用他们听得懂的话说明结果。

用户可能不知道 CLI、API key、构建命令、JSON、build_id 是什么。除非用户主动问，否则不要把这些细节作为主要解释。技术细节是 Agent 内部执行规则，不是用户操作说明。

## 什么时候使用

- 用户说“帮我发布到 RobotX”“把这个网站上线”“部署一下”“给我一个可访问链接”。
- 用户想把本地应用、静态站点、Agent 主页、作品页或 CI 产物发布到 RobotX。
- 用户想查看 RobotX 项目状态、历史版本，或把某个版本发布到正式链接。
- 用户技术背景弱，需要 Agent 代办部署并翻译错误。

## 什么时候不要使用

- 修改 RobotX 服务端、前端、Edge、Router 代码。
- 直接调用 `/api/runtime/v1/*` 跑模型或任务。
- 编写 Agent 主页、日记、作品页内容；这种场景先用 `agent-pages`，最后再用本技能发布。
- MCP 集成；`robotx mcp` 当前只是占位。
- 查看远程构建日志；`robotx logs` 和 `status --logs` 当前不可用。

## 面向用户的交互原则

- 默认替用户做，不要把用户变成命令执行员。
- 先自动判断项目路径、项目名、构建方式；只有缺关键信息时才问。
- 先判断这是“新建一个网站”还是“更新这个文件夹之前发布过的网站”；有覆盖旧网站风险时必须确认。
- 用户只要求修改、优化、预览页面时，不要自动发布到线上；只有用户说“发布、上线、更新线上、换正式链接”时才发布。
- 一次最多问 3 个短问题，优先问用户能回答的问题。
- 不要求用户理解 `--output json`、`stdout`、`build_id`、`output-dir`、`target`、`project-id`、`npm ci`。
- `--help` 输出只给 Agent 判断当前 CLI 能力，禁止原样转述给低技术用户。
- 技术错误要翻译成人话，并给出下一步。
- 结果汇报优先给链接和状态，技术 ID 放在后面或简短列出。
- 不要输出、保存或要求用户公开 API key、token、cookie、设备码。

可问的问题示例：

- “要发布当前目录，还是另一个文件夹？”
- “这是新建一个网站，还是更新之前发布过的网站？”
- “这个项目想叫什么名字？我会自动转成可用的小写英文名。”
- “这次要正式发布，还是只生成预览链接？”

不要这样问：

- “你的 output-dir 是什么？”
- “你的 target 是什么？”
- “请提供 API key。”
- “是否要用 `--local-build=true --output json`？”

## 用户可见流程

对低技术用户，用这个流程沟通：

1. 我先检查发布工具和登录状态。
2. 我会判断这是新网站，还是更新这个文件夹已经发布过的网站。
3. 我会在本地构建项目并上传发布产物。
4. 发布后我会验证状态。
5. 最后给你预览链接或正式链接。

如果需要登录，说：

```text
需要先登录 RobotX。我会打开一个登录链接，你完成授权后我继续发布。
```

如果不能自动打开浏览器，说：

```text
我会给你一个登录链接和验证码。你打开链接完成登录后，我会继续检查发布状态。
```

## Agent 内部执行规则

以下规则给 Agent 执行，不要原样讲给低技术用户。

### 1. 检查 CLI

```bash
which robotx || which robotx_cli
```

同时检查当前安装版本和命令能力：

```bash
robotx --version
robotx deploy --help
robotx targets --help
robotx access --help
robotx doctor --help
robotx --help
```

如果本机安装的 CLI 和项目源码说明不一致，优先相信当前命令的实际 `--help` 输出和本地源码，不要凭旧记忆猜参数。当前 CLI 可能没有 `domain`、`access`、`verify-url` 之类命令；没有命令时不要承诺能直接绑定自有域名或修改访问策略。

如果缺失，优先安装 release 二进制：

```bash
curl -fsSL https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh | bash
```

大陆网络或 GitHub 下载慢时，不要反复等待 `latest`。默认使用 `mr.robotx.xin` 中转并固定版本：

```bash
curl -fsSL https://mr.robotx.xin/https://raw.githubusercontent.com/haibingtown/robotx_cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.4 ROBOTX_GITHUB_PROXY=https://mr.robotx.xin bash
```

如果有内部 release 镜像：

```bash
curl -fsSL https://<mirror>/haibingtown/robotx_cli/main/scripts/install.sh \
  | env ROBOTX_VERSION=v0.4 \
      ROBOTX_DOWNLOAD_BASE_URL=https://<mirror>/haibingtown/robotx_cli/releases/download \
      bash
```

只有明确需要源码安装时，才使用：

```bash
go install github.com/haibingtown/robotx_cli/cmd/robotx@latest
```

### 2. 认证预检

任何 API 命令前先检查认证：

```bash
robotx projects --limit 1 --output json
```

如果出现 `missing_base_url`、`missing_api_key`、`401`、`403`：

- 本地交互环境：运行 `robotx login --base-url https://robotx.xin`，用户授权后重试。
- 远程或无浏览器环境：运行 `robotx login --base-url https://robotx.xin --no-browser`，让用户打开链接完成授权。
- CI 或非交互环境：不要要求用户临时粘贴密钥到聊天里；说明需要在 CI secret 或环境变量里配置 `ROBOTX_BASE_URL` 和 `ROBOTX_API_KEY`。
- 手动配置只作为 fallback，可写 `~/.robotx.yaml`：

如果出现 `proxyconnect`、`operation not permitted`、DNS/network sandbox 之类错误，这通常是当前执行环境没有网络权限，不是 RobotX 登录失效。先申请网络/沙箱权限后重试同一条命令，不要马上让用户重新登录。

```yaml
base_url: https://your-robotx-server.com
api_key: your-api-key
```

### 3. 项目名处理

RobotX 项目名必须是 DNS-safe：4-63 位，只能包含小写字母、数字、连字符，且首尾必须是字母或数字。

- 用户给中文名或带空格名称时，生成一个小写英文 slug，并告诉用户“我会用 `xxx` 作为发布项目名”。
- 不要让用户自己理解 DNS-safe。
- 如果无法安全推断，问用户想用的英文项目名。

好的项目名：`my-app`、`sanwan-demo`。  
不合格项目名：`My App`、`app`、`my_app`。

### 4. 新建/更新判断

RobotX CLI 会在项目工作区写入 `.robotx/targets.json`，记录“这个本地文件夹发布到哪个 RobotX 项目”。这份记录是给 Agent 和 CLI 用的，不要把它作为用户必须理解的概念。

默认判断：

- 当前工作区已有发布记录：默认更新这个记录对应的网站。
- 当前工作区没有发布记录：默认新建网站。
- 用户明确说“新建一个”“不要覆盖之前的”：必须新建，不能复用同名旧项目。
- 用户明确说“更新之前那个”“覆盖线上”：优先更新已有发布记录。
- 同一个工作区有多个发布目标时，用用户能懂的话询问：“这个文件夹之前发布过多个网站，要更新哪一个？”

内部目标名策略：

- 默认目标名通常用 `main`，但“主站/正式站”不一定等于 `main`；先看 `robotx targets` 的 `default_target`、`production_url` 和项目名再判断。
- 只有工作区没有任何旧记录、且用户是在新建第一个网站时，才可以默认用 `main` 作为新网站记录名。
- 如果用户要一个文件夹发布多个网站，使用简短目标名，例如 `official`、`preview`、`client-a`。
- 不要把 `target` 翻译成用户必须学习的新概念；对用户说“这个网站记录”或“要更新的网站”。
- `.robotx/targets.json` 必须跟着项目工作区走，不要写到 Codex 启动目录，除非启动目录就是项目工作区。
- 如果用户从 `dist/`、`build/`、`out/` 目录启动，CLI 会把记录写到它们的父项目目录；必要时用 `--workspace-root` 指定真正项目根。
- CLI 只会持久保存工作区内的来源路径；如果这次发布源来自 `/private/tmp` 等工作区外临时目录，本地网站记录不会保存这个临时路径。下次更新时必须重新给出真实项目路径，不能依赖当前目录或临时目录。
- 如果显式使用 `--project-id`，但没有同时指定 `--target`，这只是一次性更新远端项目，CLI 不会改写本地默认网站记录；不要把它当成绑定记录。
- 如果显式使用 `--target` 但这个记录不存在，且工作区已经有别的网站记录，除非用户明确说“新建”，否则不要继续发布。
- 如果用户说“不要主站记录”“删掉这个记录”“换另一个网站更新”，先用 `robotx targets` 查看记录，再根据记录名用 `robotx targets remove <记录名>` 删除本地记录；不要把“主站”机械翻译成 `main`，也不要手写或手改 `.robotx/targets.json`。
- `robotx targets remove` 只删除本地发布记录，不删除远端 RobotX 项目。对用户要说清楚“旧网站还在，只是不再作为这个文件夹的默认更新目标”。

重名策略：

- 新建网站时，如果服务端返回 `name_conflict`，说明这个名字已经对应一个旧 RobotX 项目；不要自动更新旧项目。
- 新版 CLI 在 `--create` 主流程里会先查询当前账号已有项目；即使服务端还没上线新的冲突策略，也会在本地拦住同名项目。
- 这时给用户两个简单选择：“换一个新名字发布成新网站”或“确认更新已有网站”。
- 只有用户明确要兼容旧行为或明确要按名字复用时，才使用 `--upsert`；`--upsert` 必须配 `--name`，不会读取或写入本地网站记录，也不能和 `--target` 混用。

### 5. 部署

新建网站的内部命令：

```bash
robotx deploy . \
  --create \
  --target <新网站记录名> \
  --name my-app \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

如果用户明确要求“公开可访问、未登录可直接打开”，新建或更新时优先加：

```bash
--access open --verify-url
```

如果项目已经发布成功，只是需要改成未登录可访问，使用：

```bash
robotx access open --project-id proj_123 --output json
```

更新本文件夹已记录网站的内部命令：

```bash
robotx deploy . \
  --update \
  --target <已有网站记录名> \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

如果只需要预览，把 `--publish` 改成 `false`：

```bash
robotx deploy . \
  --create \
  --target <新网站记录名> \
  --name my-app \
  --local-build=true \
  --publish=false \
  --wait=true \
  --output json
```

发布前先判断项目形态，Agent 自己决定产物目录，不要问低技术用户 `output-dir`。

| 项目形态 | 判断方式 | 命令策略 |
| --- | --- | --- |
| 单文件或根目录静态站 | 有 `index.html`，没有 `package.json`、`src/`、`vite.config*`、`next.config*` | 加 `--output-dir .` |
| 已构建静态产物 | 有 `dist/` | 加 `--output-dir dist` |
| CRA 或 build 产物 | 有 `build/` | 加 `--output-dir build` |
| Next 静态导出 | 有 `out/` | 加 `--output-dir out` |
| Node 前端项目 | 有 `package.json` 和 build script | 先本地构建，再按框架或产物目录判断 |
| 不确定 | 先读取项目文件 | 不要先问用户技术参数 |

单文件或根目录静态站必须这样发布：

```bash
robotx deploy . \
  --create \
  --target <新网站记录名> \
  --name my-app \
  --output-dir . \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

如果项目不是默认 Node 构建，先从项目文件推断；推断不了再使用显式参数：

```bash
robotx deploy . \
  --create \
  --target <新网站记录名> \
  --name my-app \
  --install-command "npm ci" \
  --build-command "npm run build" \
  --output-dir dist \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

CI 中建议补充版本来源：

```bash
robotx deploy . \
  --update \
  --target <已有网站记录名> \
  --version-label v1.2.3 \
  --source-ref "tag:v1.2.3@<sha>" \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

兼容旧行为的内部命令，只在用户明确要“按名字复用/更新旧项目”时使用：

```bash
robotx deploy . \
  --upsert \
  --name my-app \
  --no-write-target \
  --local-build=true \
  --publish=true \
  --wait=true \
  --output json
```

### 6. 成功判定

部署任务不算完成，直到满足：

- 已使用 `--output json`。
- 已解析 JSON，而不是只看进度日志。
- `success == true`。
- 已获得 `project_id`、`build_id`、`build_status`。
- 新建或更新后已写入本地发布记录，除非明确使用了 `--no-write-target`。
- 当 `--wait=true` 时，`build_status == "success"`。
- 如果正式发布，拿到 `production_url`，或明确说明服务端未返回正式链接。
- 如果只做预览，拿到 `preview_url`，或明确说明服务端未返回预览链接。

必要时二次确认：

```bash
robotx status --project-id proj_123 --build-id build_456 --output json
```

把“发布成功”和“访问成功”分开判断：

- 发布成功：以 `robotx deploy --output json` 和 `robotx status --output json` 为准。
- 匿名访问成功：优先用 `robotx deploy --access open --verify-url --output json` 的 `access_check` 判断；旧 CLI 才用浏览器或 `curl` 检查链接能否未登录打开。
- 如果 `status` 是 success，但 `production_url` 或 `preview_url` 返回 401/403，这不是部署失败，而是 RobotX 应用访问层需要登录、成员权限、邀请码或签名链接。
- 如果用户要求未登录可访问，发布成功后 401/403 的下一步是 `robotx access open --project-id ... --output json`，不是重新部署。
- 预览链接通常更偏向登录态验证；预览 401 时不要重部署，先说明访问策略。
- 对低技术用户汇报时，说“发布成功，但这个链接需要 RobotX 登录后访问”，不要说“发布失败”。

## 给用户的结果汇报

优先用这种格式：

```text
发布完成。

正式链接：https://...
预览链接：https://...
状态：成功
访问方式：需要 RobotX 登录后访问/可直接打开/暂未确认

我也记录了这次发布信息：
- 网站记录：main
- 项目：my-app
- 构建：build_456
```

如果只是预览：

```text
预览发布完成。

预览链接：https://...
状态：成功

这次还没有发布到正式链接。
```

如果发布成功但未登录访问被拦截：

```text
发布完成。

正式链接：https://...
状态：发布成功
访问提醒：这个 RobotX 应用当前需要登录后访问，未登录打开可能会跳到登录页或返回 401。这不是页面发布失败。
```

如果失败：

```text
这次还没有发布成功。

原因：项目没有生成可上传的发布文件夹。
下一步：我会检查构建命令和输出目录，然后重试。
```

## 常用内部命令

列项目：

```bash
robotx projects --limit 50 --output json
```

列构建版本：

```bash
robotx versions --project-id proj_123 --limit 20 --output json
```

`versions` 也可以写成：

```bash
robotx builds --project-id proj_123 --limit 20 --output json
```

查状态：

```bash
robotx status --project-id proj_123 --output json
robotx status --build-id build_456 --output json
robotx status --project-id proj_123 --build-id build_456 --output json
```

发布指定构建：

```bash
robotx publish --project-id proj_123 --build-id build_456 --output json
```

查看或修改访问策略：

```bash
robotx access status --project-id proj_123 --output json
robotx access open --project-id proj_123 --output json
robotx access login --project-id proj_123 --output json
robotx access private --project-id proj_123 --output json
```

查看本地网站记录：

```bash
robotx targets --output json
```

删除本地网站记录：

```bash
robotx targets remove <记录名> --output json
```

删除记录后如果只剩一个网站记录，CLI 会自动把它设为默认更新目标。

## 错误翻译和处理

| 技术现象 | 给用户的说法 | 下一步 |
| --- | --- | --- |
| `missing_base_url` | “还没有配置 RobotX 服务地址。” | 本地登录，或让 CI 配置环境变量 |
| `missing_api_key` | “还没有登录 RobotX。” | 本地运行登录流程；CI 使用 secret |
| API 命令里的 `401` / `403` | “登录状态失效，或者当前账号没有权限发布。” | 重新登录或换有权限的账号 |
| 链接访问返回 `401`，但 `status` 是 success | “发布成功，但这个链接需要 RobotX 登录后访问。” | 如果用户要未登录可访问，执行 `robotx access open --project-id ...`；否则说明访问策略 |
| 预览链接返回 `401` | “预览链接需要登录态验证。” | 不要重部署；用正式发布状态和 `status` 判断部署是否成功 |
| `proxyconnect` / `operation not permitted` | “当前执行环境暂时没有网络权限。” | 申请网络/沙箱权限后重试原命令 |
| `invalid_project_name` | “项目名格式不符合发布要求。” | 自动转成小写英文短横线名称，必要时问用户确认 |
| `name_conflict` | “这个网站名字已经被用过了。为了避免覆盖旧网站，我先停下来确认。” | 让用户选“换新名字”或“确认更新已有网站” |
| `target_required` | “这个文件夹之前发布过多个网站，我需要知道要更新哪一个。” | 用用户可理解的名称列出候选网站，不讲 CLI 参数 |
| `target_not_found` | “这个本地网站记录不存在，可能已经删过了。” | 重新查看本地网站记录；必要时让用户选择还存在的网站 |
| `target_exists` | “这个网站记录已经存在，不能当作新网站再创建一次。” | 如果要覆盖就更新；如果要新建就换一个网站记录和项目名 |
| `target_project_mismatch` | “这个网站记录已经绑定到另一个 RobotX 项目。” | 不要自动改绑；让用户确认是换记录、新建记录，还是先删除旧记录 |
| `missing_source_path` | “这个网站记录没有保存可用的项目文件夹位置。” | 让用户给出真实项目目录，或从真实项目目录重新发布一次 |
| `invalid_source_path` | “这个网站记录里的项目文件夹位置不安全或不可用。” | 不要继续发布；让用户给出真实项目目录，或从真实项目目录重新发布一次 |
| `missing_project_target` | “我找不到要更新的旧网站记录。” | 让用户选择已有网站，或改成新建 |
| `RobotX no longer supports remote build` | “RobotX 现在需要在本地完成构建后再上传。” | 移除 `--local-build=false`，本地构建 |
| `output directory missing: .../dist` 且根目录有 `index.html` | “这是单文件静态站，不需要 dist 目录。” | 自动用 `--output-dir .` 重试一次 |
| `output directory missing` | “项目没有生成可发布文件夹。” | 检查构建命令和产物目录；不要先问用户技术参数 |
| `build_failed` | “本地构建失败，所以还不能发布。” | 查看本地构建输出，修复依赖或脚本 |
| `robotx logs` 不可用 | “RobotX 现在不提供远程构建日志。” | 使用本地构建输出和 `status` |
| `robotx mcp` 不可用 | “当前不能用 MCP 模式发布。” | 使用 CLI + JSON |

失败时不要用大段技术日志淹没用户。先给一句人话原因，再给一个可执行下一步。

## 和其他技能的关系

- `agent-pages`：负责页面内容、结构、文案和公开展示。
- 本技能：负责把内容或应用可靠发布到 RobotX，并验证结果。

当任务是维护 Agent 公开页面时，先用 `agent-pages` 更新内容，再用本技能完成发布、验证和汇报。

## 信息源策略

如果 CLI 行为、参数或输出字段可能变化，先读取项目内这些文件确认：

- `README.md`
- `action.yml`
- `sdk/deploy-api.md`
- `cmd/deploy.go`
- `cmd/targets.go`
- `pkg/client/client.go`

如果问题涉及“为什么静态站会找 dist”“为什么链接 401”“public 是否等于匿名公开”，还要读取 RobotX 主项目源码里的这些事实源：

- `server/internal/core/service/zip_scanner.go`
- `server/internal/core/model/project.go`
- `server/internal/core/model/access_policy.go`
- `edge/auth.js`

不要凭旧记忆猜测最新部署契约。
