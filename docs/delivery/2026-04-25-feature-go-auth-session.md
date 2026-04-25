# 任务交付：auth-session

## 任务背景

根据 `task-02-auth-session`，实现管理员账号密码登录、Cookie Session、鉴权中间件和初始管理员自动初始化能力。

## 实现方案

- 新增管理员仓储和认证服务
- 使用 `bcrypt` 校验密码
- 使用 `gorilla/sessions` 管理 Cookie Session
- 启动阶段根据配置自动初始化管理员账号
- 新增认证接口并接入受保护路由

## 文件变更

- 新增 `internal/repo/admin_repo.go`
- 新增 `internal/service/auth_service.go`
- 新增 `internal/service/auth_service_test.go`
- 新增 `internal/middleware/auth_middleware.go`
- 新增 `internal/handler/auth_handler.go`
- 新增 `internal/dto/auth.go`
- 新增 `internal/bootstrap/session.go`
- 新增 `internal/bootstrap/admin_bootstrap.go`
- 修改 `internal/app/module.go`
- 修改 `internal/config/*`
- 修改 `config/config.toml.example`

## 配置与依赖变更

- 新增依赖：`github.com/gorilla/sessions`
- 新增配置：
  - `session.secret`
  - `session.cookie_name`
  - `session.max_age`
  - `session.secure`
  - `bootstrap.init_admin_username`
  - `bootstrap.init_admin_password`
- 新增环境变量：
  - `SESSION_SECRET`
  - `SESSION_COOKIE_NAME`
  - `SESSION_MAX_AGE`
  - `SESSION_SECURE`
  - `INIT_ADMIN_USERNAME`
  - `INIT_ADMIN_PASSWORD`

## 新增接口

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/profile`

## 测试结果

- 已执行 `go mod tidy`
- 已执行 `go test ./...`
- 已新增 `internal/service/auth_service_test.go`

## 风险与后续建议

- 当前 Session 使用签名 Cookie，适合单机自用场景
- 当前仅完成认证骨架，后续新增受保护接口时需统一挂接鉴权中间件
- 建议下一步进入任务 `03`，实现 SSH Key 管理和密钥加密存储
