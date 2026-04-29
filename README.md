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

建议初始化方式：

```bash
cp config/config.toml.example config/config.toml
```

然后按你的环境修改对应内容。如需本地私有覆盖项，可自行创建 `config/private.toml`；该文件适合存放敏感配置，不建议提交到仓库。

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

- `init_admin_username`：首次启动时初始化管理员用户名；留空时默认使用 `admin`
- `init_admin_password`：首次启动时初始化管理员密码；留空时自动生成随机密码并打印到启动日志

#### `[restore]`

- `mode`：恢复模式，忘记管理员密码时可临时设置为 `admin-password`
- `token`：恢复模式使用的一次性强随机 Token，启动后 5 分钟内有效，建议使用 `openssl rand -hex 32` 生成

说明：

- `bootstrap.init_admin_username` 和 `bootstrap.init_admin_password` 必须同时配置，或同时留空；同时留空时会自动创建 `admin` 并在首次创建成功后输出随机密码到日志
- 使用自动生成的随机密码首次登录后，建议立即修改管理员密码
- `session.secret`、`security.app_master_key`、初始化管理员密码、恢复 Token 等敏感信息，建议优先放在 `config/private.toml` 或环境变量中
- 游客只读模式由管理员在控制台操作菜单中开启或关闭，不再通过环境变量或配置文件控制

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

如果已经 clone 了源码，`docker-compose.yml` 在项目根目录。Compose 会启动两个服务：

- `postgres`：PostgreSQL 16，数据持久化到 `postgres_data` volume
- `app`：`traffic-monitor` 应用，前端会在镜像构建阶段打包并嵌入 Go 二进制

### 快速启动脚本

如果是在一台新机器上快速启动，不需要提前准备源码仓库或 `docker-compose.yml`，直接执行下面这段脚本即可。脚本会创建 `traffic-monitor` 目录，生成 `.env` 和 `docker-compose.yml`，然后使用 `latest` 镜像启动服务。

```bash
set -eu

mkdir -p traffic-monitor
cd traffic-monitor

if [ ! -f .env ]; then
  INIT_ADMIN_PASSWORD="$(openssl rand -hex 12)"
  cat > .env <<EOF
APP_PORT=8086
POSTGRES_PORT=5432
POSTGRES_DB=traffic_monitor
POSTGRES_USER=traffic_monitor
SESSION_SECRET=$(openssl rand -hex 32)
APP_MASTER_KEY=$(openssl rand -base64 32)
INIT_ADMIN_USERNAME=admin
INIT_ADMIN_PASSWORD=${INIT_ADMIN_PASSWORD}
POSTGRES_PASSWORD=$(openssl rand -hex 16)
EOF
  echo "Generated .env with admin password: ${INIT_ADMIN_PASSWORD}"
fi

if [ ! -f docker-compose.yml ]; then
  cat > docker-compose.yml <<'EOF'
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-traffic_monitor}
      POSTGRES_USER: ${POSTGRES_USER:-traffic_monitor}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-traffic_monitor} -d ${POSTGRES_DB:-traffic_monitor}"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "${POSTGRES_PORT:-5432}:5432"

  app:
    image: sunliang711/traffic-monitor:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      APP_ENV: production
      HTTP_ADDR: ":8086"
      POSTGRES_DSN: host=postgres user=${POSTGRES_USER:-traffic_monitor} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB:-traffic_monitor} port=5432 sslmode=disable
      SESSION_SECRET: ${SESSION_SECRET}
      APP_MASTER_KEY: ${APP_MASTER_KEY}
      INIT_ADMIN_USERNAME: ${INIT_ADMIN_USERNAME}
      INIT_ADMIN_PASSWORD: ${INIT_ADMIN_PASSWORD}
      RESTORE_MODE: ${RESTORE_MODE:-}
      RESTORE_TOKEN: ${RESTORE_TOKEN:-}
      LOG_LEVEL: ${LOG_LEVEL:-info}
      LOG_FORMAT: ${LOG_FORMAT:-json}
      COLLECTOR_INTERVAL: ${COLLECTOR_INTERVAL:-300s}
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O - http://127.0.0.1:8086/healthz || exit 1"]
      interval: 300s
      timeout: 5s
      start_period: 20s
      retries: 3
    ports:
      - "${APP_PORT:-8086}:8086"

volumes:
  postgres_data:
EOF
fi

docker compose pull
docker compose up -d
docker compose ps
echo "Web UI: http://127.0.0.1:8086"
```

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
| `APP_PORT` | 否 | 否 | `8086` | 宿主机暴露的 Web 端口 |
| `POSTGRES_PORT` | 否 | 否 | `5432` | 宿主机暴露的 PostgreSQL 端口 |
| `POSTGRES_DB` | 否 | 否 | `traffic_monitor` | PostgreSQL 数据库名 |
| `POSTGRES_USER` | 否 | 否 | `traffic_monitor` | PostgreSQL 用户名 |
| `POSTGRES_PASSWORD` | 是 | 是 | `traffic_monitor` | PostgreSQL 密码，生产环境必须修改 |
| `SESSION_SECRET` | 是 | 是 | 无 | Session 签名密钥，建议使用 `openssl rand -hex 32` 生成 |
| `APP_MASTER_KEY` | 是 | 是 | 无 | 敏感数据加密主密钥，必须是 base64 编码后的 32 字节密钥 |
| `INIT_ADMIN_USERNAME` | 否 | 否 | `admin` | 首次启动时初始化管理员用户名 |
| `INIT_ADMIN_PASSWORD` | 否 | 是 | 自动生成 | 首次启动时初始化管理员密码；未配置时会生成随机密码并打印到启动日志 |
| `RESTORE_MODE` | 否 | 否 | 无 | 忘记管理员密码时临时设置为 `admin-password` |
| `RESTORE_TOKEN` | 启用恢复模式时需要 | 是 | 无 | 恢复模式使用的一次性 Token，启动后 5 分钟内有效，建议使用 `openssl rand -hex 32` 生成 |
| `LOG_LEVEL` | 否 | 否 | `info` | 日志级别，例如 `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | 否 | 否 | `json` | 日志格式，支持 `json` / `console` |
| `COLLECTOR_INTERVAL` | 否 | 否 | `300s` | 定时采集周期 |

说明：

- `APP_MASTER_KEY` 一旦用于加密 SSH Key 等敏感数据，后续不能随意更换，否则已有密文无法解密。
- `INIT_ADMIN_USERNAME` 和 `INIT_ADMIN_PASSWORD` 必须同时配置或同时留空；同时留空时使用 `admin` 和启动日志中的随机密码首次登录。
- 使用自动生成的随机密码首次登录后，建议立即修改管理员密码。
- `RESTORE_MODE` 仅用于恢复密码，`RESTORE_TOKEN` 在服务启动后 5 分钟内有效；重置成功后应删除 `RESTORE_MODE` 和 `RESTORE_TOKEN` 并重启服务。
- `POSTGRES_DSN` 在 `docker-compose.yml` 中会根据 PostgreSQL 配置自动拼接，通常不需要在 `.env` 中手动配置。
- 如果只允许本机访问数据库，可以设置 `POSTGRES_PORT=127.0.0.1:5432`，或移除 `postgres` 服务的 `ports` 配置。

### 2. 构建并启动

首次部署或代码更新后执行：

```bash
docker compose up -d --build
```

### 管理员密码恢复

忘记管理员密码时，可以临时启用恢复模式：

```bash
RESTORE_TOKEN="$(openssl rand -hex 32)"
printf '\nRESTORE_MODE=admin-password\nRESTORE_TOKEN=%s\n' "${RESTORE_TOKEN}" >> .env
docker compose up -d
echo "Restore Token: ${RESTORE_TOKEN}"
```

然后访问 Web UI，首页会切换为管理员密码恢复页面。重置成功后删除 `.env` 中的 `RESTORE_MODE` 和 `RESTORE_TOKEN`，再执行：

```bash
docker compose up -d
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

- 管理后台：`http://127.0.0.1:8086`
- 健康检查：`http://127.0.0.1:8086/healthz`
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
image: sunliang711/traffic-monitor:latest
```

然后启动：

```bash
docker compose up -d
```

### 5. 部署后验证

```bash
curl http://127.0.0.1:8086/healthz
```

正常响应示例：

```json
{"code":200,"data":{"app_name":"traffic-monitor","db":"up","env":"production","status":"ok"},"message":"success"}
```

## 关键说明

- 服务启动时会自动执行数据库 migrate
- 首次启动会根据环境变量、配置文件或自动生成的随机密码初始化管理员
- 前端会在 Docker 构建阶段打包并嵌入 Go 二进制
- `APP_MASTER_KEY` 必须是 base64 编码后的 32 字节密钥
- `config/private.toml` 适合存放本地私有配置，不建议提交真实敏感信息
