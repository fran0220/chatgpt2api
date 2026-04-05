# ChatGPT2API

将 ChatGPT 网页版图片生成能力转为 OpenAI 兼容 API。无需 DALL-E API Key，直接复用 ChatGPT Plus/Team 账号的图片生成额度。

## 功能特性

- **OpenAI 兼容 API** — `POST /v1/images/generations` 和 `/v1/images/edits`，可直接对接支持 OpenAI 格式的客户端
- **Token 池管理** — 支持多个 ChatGPT Access Token 轮询，最少使用���先策略均匀分摊负载
- **自动容错** — 连续失败自动进入冷却状态，避免反复使用失效 Token
- **图片缓存** — 生成的图片自动下载缓存到本地，通过自有域名提供稳定访问
- **管理面板** — Web UI 管理 Token（添加/导入/删除/状态���看）和配置 API Key
- **PoW 自动求解** — 自动完成 ChatGPT 的 Proof-of-Work 验证
- **支持图片编辑** — 支持 transformation（整图重新生成）和 inpainting（局部编辑）

## 快速开始

### Docker 部署（推荐）

```bash
# 克隆仓库
git clone https://github.com/fran0220/agent-skills.git
cd agent-skills/chatgpt2api

# 启动服务
docker compose up -d --build

# 服务运行在 http://localhost:8200
```

### 本地运行

```bash
cd chatgpt2api

# 可选：通过环境变量注入初始 Token
export ACCESS_TOKEN="your-chatgpt-access-token"

go run .
# 默认监听 :8080
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_HOST` | `0.0.0.0` | 监听地址 |
| `SERVER_PORT` | `8080` | 监听端口（Docker 内默认 `8200`） |
| `ACCESS_TOKEN` | — | 初始 ChatGPT Access Token（仅在 Token 池为空时生效） |
| `TZ` | `Asia/Shanghai` | 时区 |

### 配置文件

默认配置 `config.defaults.toml`，用户自定义配置写入 `data/config.toml`（通过管理面板或手动创建）：

```toml
[app]
api_key = ""              # API 调用密钥（逗号分��支持多个，留空=无需认证）
app_key = "chatgpt2api"   # 管理后台密码
image_format = "url"      # 图片默认返回格式 (url / b64_json)

[chatgpt]
model = "gpt-5-3"         # 图片生成模型
sse_timeout = 300          # SSE 流超时（秒）
poll_interval = 3          # 异步轮询间隔（秒）
poll_max_wait = 180        # 异步轮询最大等待（秒）

[token]
fail_threshold = 5         # 连续失败阈值（达到后自动冷却）
```

## 管理面板

访问 `http://localhost:8200/admin` 进入管理后台（默认密码 `chatgpt2api`）。

### 功能

- **Token 管理** — 查看所有 Token 状态（正常/限流/失效���、调用次数、添加/批量导入/删除
- **API Key 设置** — ���线配置服务访问密钥，保存后立即生效无需重启
- **统计卡片** — 总数、正常、限流、失效的 Token 数量一目了然
- **状态筛选** — 按状态过滤 Token 列表

### Token 获取方式

Token 是 ChatGPT 网页版的 Access Token（JWT），获取方法：

1. 登录 [chatgpt.com](https://chatgpt.com)
2. 打开浏览器开发者工具（F12）
3. 访问 `https://chatgpt.com/api/auth/session`
4. 复制响应中的 `accessToken` 字段

> ⚠️ Access Token 有效期约 2 周，过期后需要重新获取。

## API 端点

### 图片生成

```
POST /v1/images/generations
Authorization: Bearer <api_key>  # 如果配置了 api_key
```

```json
{
  "prompt": "a cute cat on a space station, watercolor style",
  "n": 1,
  "size": "1024x1024",
  "quality": "standard",
  "response_format": "url"
}
```

响应：

```json
{
  "created": 1775366000,
  "data": [
    {
      "url": "https://your-domain.com/v1/files/image/abc123.png",
      "revised_prompt": "A cute orange tabby cat floating..."
    }
  ]
}
```

### 图片编辑

```
POST /v1/images/edits
Authorization: Bearer <api_key>
```

```json
{
  "prompt": "change the cat to a golden retriever",
  "image_file_id": "file_00000000dc7c71f5b1283eba162ff266",
  "gen_id": "beb94ced-6569-4438-bced-01ff1f571a24",
  "conversation_id": "69d1edac-bc34-83e8-adab-534411adf767",
  "parent_message_id": "8e9ca440-66cc-46fe-8ee1-e0822a6adacb"
}
```

> 添加 `mask_file_id` 参数可进行局部编辑（inpainting），否���为整图变换（transformation）。

### 模型列表

```
GET /v1/models
Authorization: Bearer <api_key>
```

### 管理 API

所有管理端点需要 `Authorization: Bearer <app_key>` 认证。

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/admin/verify` | POST | 验证管理密码 |
| `/v1/admin/tokens` | GET | 获取 Token 列表和统计 |
| `/v1/admin/tokens` | POST | 添加 Token `{"token":"...","note":"..."}` |
| `/v1/admin/tokens` | DELETE | 删除 Token `{"token":"..."}` |
| `/v1/admin/config` | GET | 获取当前配置（API Key 脱敏显示） |
| `/v1/admin/config` | PUT | 更新配置 `{"api_key":"..."}` |

### 其他

| 端点 | 说明 |
|------|------|
| `GET /health` | 健康检查 |
| `GET /v1/files/image/{filename}` | 访问缓存的图片文件 |

## 项目结构

```
chatgpt2api/
├── main.go                          # 入��：配置加载、Token 初始化、HTTP 服务
├── config.defaults.toml             # 默认配置
├── Dockerfile                       # 多阶段构建
├── docker-compose.yml               # Docker Compose 编排
│
├── api/
│   ├── router.go                    # 路由注册（API + 管理 + 页面）
│   ├── image.go                     # 图片生成/编辑处理器
│   ├── admin.go                     # Token 管理 + 配置 API
│   ├── files.go                     # 缓存图片文件服务
│   ├── models.go                    # 模型列表
│   ├── pages.go                     # 静态页面服务（管理面板）
│   └── response.go                  # JSON 响应工具函数
│
├── handler/
│   ├── client.go                    # ChatGPT 后端 API 客户端
│   ├── transport.go                 # Chrome 指纹 HTTP Transport
│   └── pow.go                       # Proof-of-Work 求解器
│
├── internal/
│   ├── auth/auth.go                 # Bearer Token 认证中间件
│   ├── config/config.go             # TOML 配置管理（分层加载 + 热更新）
│   ├── middleware/                   # CORS、RequestID、Logger 中间件
│   ├── storage/local.go             # Token 本地 JSON 持久化
│   └── token/
│       ├── models.go                # TokenInfo 数据结构
│       ├── pool.go                  # Token 池（最少使用优先选择）
│       └── manager.go               # Token 管理器（单例）
│
├── _public/                         # 管理面板前端
│   └── static/
│       ├── admin/pages/             # 登录页 + Token 管理页
│       ├── admin/js/                # 前端逻辑
│       ├── admin/css/               # 页面样式
│       └── common/                  # 共用 CSS/JS（认证、Toast）
│
└── data/                            # 运行时数据（Docker volume 挂载）
    ├── config.toml                  # 用户自定义配置（可选）
    ├── token.json                   # Token 持久化存储
    └── tmp/image/                   # 图片缓存目录
```

## 工作原理

```
客户端 → POST /v1/images/generations
         ↓
    认证检查 (API Key)
         ↓
    Token 池选择 (最少使用优先)
         ↓
    ChatGPT Backend API:
      1. POST /sentinel/chat-requirements → 获取验证 Token + PoW 挑战
      2. 求解 Proof-of-Work
      3. POST /conversation (SSE) → 流式接收生成状态
      4. 异步轮询直到图片生成完成
      5. GET /files/download/{file_id} → 获取下载 URL
         ↓
    下载图片 → 缓存到本地
         ↓
    返回本地缓存 URL 给客户端
```

## Nginx 反向代理

```nginx
server {
    listen 443 ssl;
    server_name chatgpt.example.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8200;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;  # 图片生成可能需要较长时间
    }
}
```

## License

MIT
