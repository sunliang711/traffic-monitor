# 游客只读模式交付说明

## 任务背景

- 当前系统默认需要管理员登录后才能进入控制台。
- 本次新增管理员可控的游客模式，开启后未登录用户只能访问监控查询类接口，写操作和敏感配置查询仍要求管理员登录。

## 实现方案

- 新增 `app_settings` 持久化设置，游客模式默认关闭，只能由管理员在控制台开启或关闭。
- 新增管理员设置接口：
  - `GET /api/v1/settings/guest-mode`
  - `PUT /api/v1/settings/guest-mode`
- 后端路由拆分为只读查询组和管理员组：
  - 游客可查询：全局阈值、单机阈值、流量样本、告警记录。
  - 管理员专属：机器列表/详情、SSH Key、通知渠道、通知代理、备份、机器写操作、阈值写操作、手动采集、历史清理。
- 前端新增游客状态：
  - 未登录且游客接口可访问时进入只读控制台。
  - 游客模式只展示样本、阈值、告警 3 个页面，隐藏总览、机器、SSH Key、通知设置、备份、历史清理、机器编辑、连接测试、手动采集、阈值保存等入口。
  - 管理员端顶部操作菜单提供游客模式开关。
  - 账号菜单保留管理员登录入口。

## 文件变更

- 新增 `internal/model/app_setting.go`
- 新增 `internal/repo/app_setting_repo.go`
- 新增 `internal/service/settings_service.go`
- 新增 `internal/handler/settings_handler.go`
- 新增 `internal/dto/settings.go`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/middleware/auth_middleware.go`
- 修改 `internal/handler/auth_handler.go`
- 修改 `internal/handler/machine_handler.go`
- 修改 `internal/handler/threshold_handler.go`
- 修改 `internal/handler/traffic_sample_handler.go`
- 修改 `internal/handler/alert_handler.go`
- 修改 `web/src/App.tsx`
- 修改 `web/src/components/OverviewTab.tsx`
- 修改 `web/src/components/ThresholdEditor.tsx`
- 修改 `web/src/pages/MachinesPage.tsx`
- 修改 `web/src/pages/SamplesPage.tsx`
- 修改 `web/src/pages/ThresholdsPage.tsx`
- 修改 `web/src/lib/i18n.tsx`
- 修改 `README.md`

## 设置变更

- 新增数据库表：`app_settings`
- 游客模式设置键：`guest_mode_enabled`
- 已移除 `GUEST_ENABLED` / `[guest]` 配置入口，游客模式不再由环境变量或配置文件控制。

## 测试结果

- `go test ./...`：通过
- `npm run build`：通过

## 风险与后续建议

- 游客模式会暴露样本、阈值和告警数据，样本与告警中的机器信息仅按机器 ID 兜底展示。
- 如后续需要更细粒度控制，建议增加独立的只读 DTO，对游客进一步限制具体字段。
