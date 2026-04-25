# 任务交付：threshold-policy

## 任务背景

根据 `task-05-threshold-policy`，实现全局阈值和单机覆盖阈值管理，支持小时/日、上行/下行/总流量三个维度，并统一按 `MB` 存储。

## 实现方案

- 新增 `global_threshold_rules` 和 `machine_threshold_rules` 两张表
- 服务层统一将 `MB/GB` 转换为 `MB`
- 单机查询时优先读取单机覆盖，缺失时回退全局阈值
- 提供全局阈值接口和单机阈值接口

## 文件变更

- 新增 `internal/model/threshold_rule.go`
- 新增 `internal/repo/threshold_rule_repo.go`
- 新增 `internal/service/threshold_service.go`
- 新增 `internal/service/threshold_service_test.go`
- 新增 `internal/handler/threshold_handler.go`
- 新增 `internal/dto/threshold.go`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/handler/auth_handler.go`

## 新增接口

- `GET /api/v1/thresholds/global`
- `PUT /api/v1/thresholds/global`
- `GET /api/v1/machines/:id/thresholds`
- `PUT /api/v1/machines/:id/thresholds`

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go test ./...`
- 已新增 `internal/service/threshold_service_test.go`

## 风险与后续建议

- 当前 `PUT` 接口允许提交部分规则，不强制必须一次提交完整 6 个维度
- 下一步建议进入任务 `06`，把采集结果和阈值判定链路接起来
