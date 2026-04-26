# traffic-monitor

`traffic-monitor` 是一个通过 `SSH + vnstat` 监控远程机器流量的服务，支持：

- 机器管理与 SSH Key 管理
- 小时 / 日流量采集
- 全局阈值与单机覆盖阈值
- `Webhook` / `Telegram` 通知
- React 管理界面

## 本地开发

后端：

```bash
go test ./...
go run ./cmd/server
```

前端：

```bash
cd web
npm install
npm run build
```

## 配置说明

项目配置按以下优先级加载：

1. 内置默认配置 `internal/config/default.toml`
2. `config/config.toml`
3. `config/private.toml`
4. 环境变量

也就是说：

- `config/config.toml` 适合放通用、可共享的项目配置
- `config/private.toml` 适合放本地覆盖项和敏感配置
- 环境变量适合在 Docker / CI / 生产环境中覆盖运行参数

### 配置文件模板

仓库中提供了以下模板文件：

- `config/config.toml.example`
- `config/private.toml.example`

建议初始化方式：

```bash
cp config/config.toml.example config/config.toml
cp config/private.toml.example config/private.toml
```

然后按你的环境修改对应内容。

### 当前配置项总览

#### `[app]`

- `name`：应用名称
- `env`：运行环境，例如 `development` / `production`

#### `[collector]`

- `enabled`：是否启用定时采集
- `interval`：采集周期
- `max_workers`：并发 worker 数
- `retry_times`：采集失败重试次数

#### `[http]`

- `addr`：HTTP 监听地址
- `read_timeout`：读取超时
- `write_timeout`：写入超时
- `stop_timeout`：服务优雅停止超时

#### `[database]`

- `dsn`：PostgreSQL 连接串
- `max_idle_conns`：最大空闲连接数
- `max_open_conns`：最大打开连接数
- `conn_max_lifetime`：连接最大生命周期
- `ping_timeout`：启动时数据库探活超时

#### `[log]`

- `level`：日志级别，例如 `debug` / `info` / `warn` / `error`
- `format`：日志格式，支持：
  - `json`
  - `console`

#### `[session]`

- `secret`：会话签名密钥
- `cookie_name`：Cookie 名称
- `max_age`：会话有效期
- `secure`：是否仅通过 HTTPS 发送 Cookie

#### `[ssh]`

- `dial_timeout`：SSH 建连超时
- `command_timeout`：SSH 命令执行超时

#### `[security]`

- `app_master_key`：用于保护敏感数据的主密钥，必须是 **base64 编码后的 32 字节密钥**

#### `[bootstrap]`

- `init_admin_username`：首次启动时初始化管理员用户名
- `init_admin_password`：首次启动时初始化管理员密码

说明：

- `bootstrap.init_admin_username` 和 `bootstrap.init_admin_password` 必须同时配置，或同时留空
- `session.secret`、`security.app_master_key`、初始化管理员密码等敏感信息，建议优先放在 `config/private.toml` 或环境变量中

## 日志格式说明

后端日志统一使用 `zerolog`，并且 `Fx` 框架日志也已经接入同一套日志输出。

可通过 `log.format` 或环境变量 `LOG_FORMAT` 控制日志格式：

### `json`

适合：

- Docker
- 日志采集系统
- ELK / Loki / Datadog 等结构化日志平台

示例：

```toml
[log]
level = "info"
format = "json"
```

### `console`

适合：

- 本地开发
- 终端直接查看
- 调试时更易读

示例：

```toml
[log]
level = "debug"
format = "console"
```

也可以通过环境变量覆盖：

```bash
LOG_LEVEL=debug
LOG_FORMAT=console
```

## Docker 部署

项目提供了多阶段 `Dockerfile` 和 `docker-compose.yml`。Compose 会启动两个服务：

- `postgres`：PostgreSQL 16，数据持久化到 `postgres_data` volume
- `app`：`traffic-monitor` 应用，前端会在镜像构建阶段打包并嵌入 Go 二进制

### 1. 准备环境变量

复制环境变量模板：

```bash
cp .env.example .env
```

至少需要修改以下敏感配置：

```bash
SESSION_SECRET=$(openssl rand -hex 32)
APP_MASTER_KEY=$(openssl rand -base64 32)
INIT_ADMIN_USERNAME=admin
INIT_ADMIN_PASSWORD=replace-with-strong-password
POSTGRES_PASSWORD=replace-with-strong-postgres-password
```

配置说明：

| 变量名 | 必填 | 敏感 | 默认值 | 说明 |
|--------|:---:|:---:|--------|------|
| `APP_ENV` | 否 | 否 | `production` | 应用运行环境 |
| `APP_PORT` | 否 | 否 | `8080` | 宿主机暴露的 Web 端口 |
| `POSTGRES_PORT` | 否 | 否 | `5432` | 宿主机暴露的 PostgreSQL 端口 |
| `POSTGRES_DB` | 否 | 否 | `traffic_monitor` | PostgreSQL 数据库名 |
| `POSTGRES_USER` | 否 | 否 | `traffic_monitor` | PostgreSQL 用户名 |
| `POSTGRES_PASSWORD` | 是 | 是 | `traffic_monitor` | PostgreSQL 密码，生产环境必须修改 |
| `SESSION_SECRET` | 是 | 是 | 无 | Session 签名密钥，建议使用 `openssl rand -hex 32` 生成 |
| `APP_MASTER_KEY` | 是 | 是 | 无 | 敏感数据加密主密钥，必须是 base64 编码后的 32 字节密钥 |
| `INIT_ADMIN_USERNAME` | 首次部署需要 | 否 | `admin` | 首次启动时初始化管理员用户名 |
| `INIT_ADMIN_PASSWORD` | 首次部署需要 | 是 | 无 | 首次启动时初始化管理员密码 |
| `LOG_LEVEL` | 否 | 否 | `info` | 日志级别，例如 `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | 否 | 否 | `json` | 日志格式，支持 `json` / `console` |
| `COLLECTOR_INTERVAL` | 否 | 否 | `300s` | 定时采集周期 |

说明：

- `APP_MASTER_KEY` 一旦用于加密 SSH Key 等敏感数据，后续不能随意更换，否则已有密文无法解密。
- `INIT_ADMIN_USERNAME` 和 `INIT_ADMIN_PASSWORD` 必须同时配置或同时留空。
- `POSTGRES_DSN` 在 `docker-compose.yml` 中会根据 PostgreSQL 配置自动拼接，通常不需要在 `.env` 中手动配置。
- 如果只允许本机访问数据库，可以设置 `POSTGRES_PORT=127.0.0.1:5432`，或移除 `postgres` 服务的 `ports` 配置。

### 2. 构建并启动

首次部署或代码更新后执行：

```bash
docker compose up -d --build
```

查看服务状态：

```bash
docker compose ps
```

查看应用日志：

```bash
docker compose logs -f app
```

启动后默认访问：

- 管理后台：`http://127.0.0.1:8080`
- 健康检查：`http://127.0.0.1:8080/healthz`
- PostgreSQL：`127.0.0.1:5432`

### 3. 常用运维命令

重启应用：

```bash
docker compose restart app
```

停止服务但保留数据卷：

```bash
docker compose down
```

停止服务并删除 PostgreSQL 数据卷：

```bash
docker compose down -v
```

更新部署：

```bash
git pull
docker compose up -d --build
```

### 4. 使用已发布镜像

如果使用 GitHub tag 发布出来的 Docker Hub 镜像，可以把 `docker-compose.yml` 中 `app` 服务的构建配置：

```yaml
build:
  context: .
  dockerfile: Dockerfile
```

替换为：

```yaml
image: sunliang711/traffic-monitor:v1.2.3
```

然后启动：

```bash
docker compose up -d
```

其中 `v1.2.3` 替换为实际发布的 tag。

### 5. 部署后验证

```bash
curl http://127.0.0.1:8080/healthz
```

正常响应示例：

```json
{"code":200,"data":{"app_name":"traffic-monitor","db":"up","env":"production","status":"ok"},"message":"success"}
```

## 关键说明

- 服务启动时会自动执行数据库 migrate
- 首次启动会根据环境变量或配置文件自动初始化管理员
- 前端会在 Docker 构建阶段打包并嵌入 Go 二进制
- `APP_MASTER_KEY` 必须是 base64 编码后的 32 字节密钥
- `config/private.toml` 适合存放本地私有配置，不建议提交真实敏感信息
