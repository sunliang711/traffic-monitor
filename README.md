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
- 首次启动会根据环境变量自动初始化管理员
- 前端会在 Docker 构建阶段打包并嵌入 Go 二进制
- `APP_MASTER_KEY` 必须是 base64 编码后的 32 字节密钥
