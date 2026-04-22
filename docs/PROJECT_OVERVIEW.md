# RobotX CLI - 项目总览

## 🎯 项目目标

将 RobotX Server 的 deploy-only 能力封装成独立的 CLI 工具，供 AI agents 使用。

## ✅ 当前状态

**核心功能已完成，可用于测试和集成**

## 📦 主要交付物

### 1. CLI 工具
- ✅ `robotx deploy` - 部署项目
- ✅ `robotx status` - 查询状态
- ⚠️ `robotx logs` - 历史命令，现已不支持（RobotX 不再提供远程构建日志）
- ✅ `robotx publish` - 发布到生产
- ✅ `robotx update` - 更新配置
- ✅ `robotx mcp` - MCP 服务器模式

### 2. 客户端库
- ✅ Python 客户端 (`examples/robotx_client.py`)
- ✅ TypeScript 客户端 (`examples/robotx_client.ts`)

### 3. 文档
- ✅ 完整使用文档 (../README.md)
- ✅ 快速入门 (QUICKSTART.md)
- ✅ AI Agent 集成指南 (AI_AGENT_INTEGRATION.md)
- ✅ 示例代码 (examples/)

## 🚀 快速开始

```bash
# 构建
make build

# 配置
cat > ~/.robotx.yaml << 'YAML'
base_url: https://your-robotx-server.com
api_key: your-api-key
YAML

# 部署
./robotx deploy ./my-app --name my-app --publish
```

## 💡 使用示例

### CLI 直接使用
```bash
robotx deploy ./my-app --name my-app --publish
```

### Python 集成
```python
from robotx_client import RobotXClient

client = RobotXClient()
result = client.deploy('./my-app', name='my-app', publish=True)
print(f"Deployed to: {result['url']}")
```

### TypeScript 集成
```typescript
import { RobotXClient } from './robotx_client';

const client = new RobotXClient();
const result = await client.deploy('./my-app', { name: 'my-app', publish: true });
console.log(`Deployed to: ${result.url}`);
```

### Claude Desktop 集成
```json
{
  "mcpServers": {
    "robotx": {
      "command": "/usr/local/bin/robotx",
      "args": ["mcp"]
    }
  }
}
```

## 📚 文档导航

| 文档 | 用途 |
|------|------|
| [README.md](../README.md) | 完整使用文档 |
| [QUICKSTART.md](QUICKSTART.md) | 5 分钟快速入门 |
| [AI_AGENT_INTEGRATION.md](AI_AGENT_INTEGRATION.md) | AI Agent 集成指南 |
| [EXAMPLES.md](EXAMPLES.md) | 使用示例 |
| [examples/README.md](../examples/README.md) | 客户端库文档 |
| [COMPLETION_REPORT.md](COMPLETION_REPORT.md) | 完成报告 |

## 🎯 核心特性

- ✅ JSON 输出格式（易于程序解析）
- ✅ 多种配置方式（文件/环境变量/参数）
- ✅ 完善的错误处理
- ✅ 异步部署支持
- ✅ MCP 协议支持
- ✅ 跨平台支持

## 🔄 典型工作流

```
AI Agent 创建项目
    ↓
RobotX CLI 打包上传
    ↓
RobotX Server 构建
    ↓
Runtime 运行
    ↓
返回 URL 给用户
```

## 📊 项目统计

- **代码文件**: 20+
- **代码行数**: ~4,700
- **支持语言**: Go, Python, TypeScript
- **文档页数**: 7 个主要文档

## 🔮 后续计划

### 短期
- [ ] 实际部署测试
- [ ] 功能增强（环境变量、实时日志等）
- [ ] 测试覆盖

### 中期
- [ ] REST API 服务器
- [ ] 更多集成示例
- [ ] 性能优化

### 长期
- [ ] 发布到包管理器
- [ ] 高级功能（多环境、回滚等）

## 📞 获取帮助

- 查看文档: [README.md](../README.md)
- 运行演示: `./demo.sh`
- 查看示例: [examples/](examples/)

---

**版本**: v1.0.0-beta  
**状态**: ✅ 可用于测试和集成  
**更新**: 2024-02-03
