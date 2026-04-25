# 任务交付：alerting-notification

## 任务背景

根据 `task-08-alerting-notification`，实现阈值判定、告警去重、通知渠道发送和投递记录。

## 实现方案

- 新增 `alerts`、`notification_channels`、`notification_deliveries` 三张表
- 样本采集入库后立即执行阈值判定
- 使用 `machine_id + period_type + metric_type + bucket_time` 生成唯一 `alert_key`
- 支持 `Webhook` 和 `Telegram` 两种通知渠道
- 无启用渠道时将告警标记为 `skipped`

## 文件变更

- 新增 `internal/model/alert.go`
- 新增 `internal/repo/alert_repo.go`
- 新增 `internal/repo/notification_repo.go`
- 新增 `internal/service/alert_service.go`
- 新增 `internal/service/alert_service_test.go`
- 新增 `internal/handler/alert_handler.go`
- 新增 `internal/dto/alert.go`
- 修改 `internal/service/traffic_collection_service.go`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/handler/auth_handler.go`

## 新增接口

- `GET /api/v1/alerts`
- `GET /api/v1/notification-channels`
- `PUT /api/v1/notification-channels/webhook`
- `PUT /api/v1/notification-channels/telegram`

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go test ./...`
- 已新增 `internal/service/alert_service_test.go`

## 风险与后续建议

- 当前通知发送没有单独的重试退避策略，后续可按渠道做更细粒度控制
- `Telegram` 渠道列表接口会返回脱敏后的 token 摘要，但仍建议前端不要重复回显敏感值
- 下一步建议进入任务 `09`，实现 React 管理界面
