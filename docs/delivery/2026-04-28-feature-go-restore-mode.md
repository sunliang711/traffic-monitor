# 任务交付：admin-password-restore-mode

## 任务背景

为忘记管理员密码的场景新增显式恢复模式。恢复模式需要启动时配置强随机 Token，避免在公网暴露无鉴权密码重置入口。

## 实现方案

- 新增 `restore.mode` 与 `restore.token` 配置，支持环境变量 `RESTORE_MODE`、`RESTORE_TOKEN`
- 当 `restore.mode = "admin-password"` 时，仅注册健康检查和恢复接口，正常管理 API 不开放
- 前端启动时检测恢复状态，恢复模式下首页渲染管理员密码重置表单
- Restore Token 启动后 5 分钟内有效，重置成功后当前进程内立即失效
- 恢复模式下跳过流量采集调度器和历史清理调度器

## 文件变更

- 修改 `internal/config/config.go`
- 修改 `internal/config/default.toml`
- 修改 `internal/config/sources_default.go`
- 修改 `internal/config/logging.go`
- 修改 `internal/dto/auth.go`
- 修改 `internal/service/auth_service.go`
- 修改 `internal/service/traffic_scheduler.go`
- 修改 `internal/service/history_cleanup_service.go`
- 新增 `internal/handler/restore_handler.go`
- 修改 `internal/handler/auth_handler.go`
- 修改 `internal/app/module.go`
- 修改 `web/src/App.tsx`
- 修改 `web/src/lib/app-types.ts`
- 修改 `web/src/lib/i18n.tsx`
- 修改 `web/src/types.ts`
- 修改 `web/src/styles.css`
- 修改 `.env.example`
- 修改 `docker-compose.yml`
- 修改 `config/config.toml.example`
- 修改 `README.md`

## 配置变更

新增配置：

```toml
[restore]
mode = ""
token = ""
```

新增环境变量：

- `RESTORE_MODE`
- `RESTORE_TOKEN`

## 新增接口

- `GET /api/v1/restore/status`
- `POST /api/v1/restore/admin-password`

## 测试结果

- 已新增配置校验测试
- 已新增管理员密码恢复 Service 测试
- 已新增恢复 Handler Token 校验与一次性使用测试

## 风险与后续建议

- 恢复模式只应短期开启，重置成功后必须删除 `RESTORE_MODE` 和 `RESTORE_TOKEN` 并重启服务
- `RESTORE_TOKEN` 应使用 `openssl rand -hex 32` 生成，禁止使用短 token；如果超过 5 分钟未完成恢复，需要重启服务重新进入恢复窗口
