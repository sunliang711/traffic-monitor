# 任务交付：scheduler-workerpool

## 任务背景

根据 `task-07-scheduler-workerpool`，实现定时调度、worker pool 并发控制和失败重试，驱动流量采集任务持续运行。

## 实现方案

- 新增 `collector` 配置段，控制是否启用调度器、采集周期、worker 数量和重试次数
- 新增 `TrafficScheduler`，通过 `Fx Lifecycle` 管理启动和停止
- 调度器按固定周期查询启用中的机器，并投递到固定 worker pool
- 单机采集失败后按配置次数重试

## 文件变更

- 新增 `internal/service/traffic_scheduler.go`
- 新增 `internal/service/traffic_scheduler_test.go`
- 修改 `internal/config/config.go`
- 修改 `internal/config/default.toml`
- 修改 `internal/app/module.go`
- 修改 `internal/service/traffic_collection_service.go`
- 修改 `config/config.toml.example`

## 配置与依赖变更

- 新增配置：
  - `collector.enabled`
  - `collector.interval`
  - `collector.max_workers`
  - `collector.retry_times`
- 无新增第三方依赖

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go test ./...`
- 已新增 `internal/service/traffic_scheduler_test.go`

## 风险与后续建议

- 当前重试退避固定为短延时，后续可按失败类型区分重试策略
- 当前调度器没有单独暴露运行状态接口，后续可补健康或状态查询
- 下一步建议进入任务 `08`，把阈值判定和通知链路接入
