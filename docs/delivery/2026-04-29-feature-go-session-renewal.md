# 任务交付：session-renewal

## 任务背景

为管理后台 Cookie Session 增加滑动续期能力：访问受保护 Admin API 时校验 Session 过期时间，剩余时间低于 TTL 一半时自动续期并重新下发 Cookie。

## 实现方案

- 登录成功后在签名 Session Cookie 中写入 `expires_at`
- 受保护 Admin API 在鉴权中间件中校验 `current_admin_id` 和 `expires_at`
- 当 `expires_at - now < session.max_age / 2` 时，将 `expires_at` 更新为 `now + session.max_age` 并保存 Session
- 前端无需变更，仍沿用现有 401 跳转登录页逻辑

## 文件变更

- 修改 `internal/handler/auth_handler.go`
- 修改 `internal/middleware/auth_middleware.go`
- 新增 `internal/handler/auth_handler_test.go`
- 新增 `internal/middleware/auth_middleware_test.go`

## 配置与依赖变更

- 未新增配置项
- 未新增依赖
- 续期 TTL 继续复用现有 `session.max_age`

## 测试结果

- `go test ./internal/handler ./internal/middleware`
- `go test ./...`

## 风险与注意事项

- 旧登录态 Cookie 中没有 `expires_at`，上线后会被视为未认证，需要重新登录
- 当前仍是 CookieStore 模式，不支持服务端强制踢下线；如后续需要多设备管理或强制失效，可再引入服务端 Session 表

## Changelog

### [Unreleased]

### Added

- Session 滑动续期：Admin API 访问时按半 TTL 阈值自动刷新 Cookie 过期时间
