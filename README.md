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

## Docker 运行

1. 复制环境文件：

```bash
cp .env.example .env
```

2. 修改以下关键变量：

- `SESSION_SECRET`
- `APP_MASTER_KEY`
- `INIT_ADMIN_USERNAME`
- `INIT_ADMIN_PASSWORD`

如需覆盖日志格式，也可以设置：

- `LOG_LEVEL`
- `LOG_FORMAT`

其中 `APP_MASTER_KEY` 可用下面的命令生成：

```bash
openssl rand -base64 32
```

`SESSION_SECRET` 可用下面的命令生成：

```bash
openssl rand -hex 32
```

3. 启动服务：

```bash
docker compose up --build
```

启动后：

- 应用地址：`http://127.0.0.1:8080`
- PostgreSQL：`127.0.0.1:5432`

## 关键说明

- 服务启动时会自动执行数据库 migrate
- 首次启动会根据环境变量或配置文件自动初始化管理员
- 前端会在 Docker 构建阶段打包并嵌入 Go 二进制
- `APP_MASTER_KEY` 必须是 base64 编码后的 32 字节密钥
- `config/private.toml` 适合存放本地私有配置，不建议提交真实敏感信息
