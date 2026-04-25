# 重构交付：config-source-chain

## 重构背景

原有配置加载使用单个 `config.go` 直接操作 `Viper`，虽具备多来源覆盖能力，但没有形成清晰的 `Source` 抽象，也将默认配置硬编码在 Go 代码中。

## 重构方案

- 引入 `Source` 接口，统一抽象配置来源
- 将默认配置迁移到内嵌 `default.toml`
- 保持配置优先级不变：默认配置 < `config.toml` < `private.toml` < 环境变量
- 保持 `NewConfig()`、`ProvideXxxConfig()` 和 `Config.Validate()` 对外行为不变

## 文件变更

- 新增 `internal/config/default.toml`
- 新增 `internal/config/loader.go`
- 新增 `internal/config/source.go`
- 新增 `internal/config/sources_default.go`
- 新增 `internal/config/config_test.go`
- 重写 `internal/config/config.go`
- 更新 `config/config.toml.example`
- 更新 `docs/project/traffic-monitor/conventions.md`

## 行为保持策略

- 继续通过 `NewConfig()` 返回最终配置对象
- 继续使用相同的配置结构体和校验逻辑
- 继续支持 `config/config.toml`、`config/private.toml` 和环境变量覆盖
- `database.dsn` 仍然是必填项，缺失时启动失败

## 测试与验证结果

- 已执行 `go mod tidy`
- 已执行 `go test ./internal/config ./...`
- 新增测试覆盖：
  - Source 优先级叠加顺序
  - 缺少必填配置时的校验失败

## 风险与后续建议

- 当前环境变量仍是显式绑定方式；后续新增配置项时要同步补充到环境变量 Source
- 如果后面要支持更多配置源，如远程配置中心，可继续新增 `Source` 实现而不改主加载流程
