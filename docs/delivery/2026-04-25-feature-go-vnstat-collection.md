# 任务交付：vnstat-collection

## 任务背景

根据 `task-06-vnstat-collection`，实现通过 SSH 执行 `vnstat --json`、解析小时/日流量并保存样本数据。

## 实现方案

- 新增 `traffic_samples` 表，按 `machine_id + period_type + bucket_time` 做样本去重
- 复用现有 SSH 能力执行 `vnstat --json -i <network_interface>`
- 解析当前网卡的小时和日流量，统一转换为 `MB`
- 提供手动触发采集接口和样本查询接口

## 文件变更

- 新增 `internal/model/traffic_sample.go`
- 新增 `internal/repo/traffic_sample_repo.go`
- 新增 `internal/service/traffic_collection_service.go`
- 新增 `internal/service/traffic_collection_service_test.go`
- 新增 `internal/dto/traffic_sample.go`
- 新增 `internal/handler/traffic_sample_handler.go`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/handler/auth_handler.go`

## 新增接口

- `GET /api/v1/traffic-samples`
- `POST /api/v1/system/collect-now`

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go test ./...`
- 已新增 `internal/service/traffic_collection_service_test.go`

## 风险与后续建议

- 当前 `vnstat` 解析按最新 JSON 结构优先处理，后续需要用真实远端环境再验证不同版本兼容性
- 当前采集只支持手动触发，下一步建议进入任务 `07`，接入调度器和并发 worker pool
