# 任务交付：project-bootstrap

## 任务背景

根据 `task-01-project-bootstrap`，完成 `traffic-monitor` 的首个开发任务，建立可运行的后端骨架、前端工程骨架和基础配置链路。

## 实现方案

- 后端采用 `Gin + Fx + GORM + Viper + Zerolog`
- 前端采用 `React + Vite`
- 后端通过 `Go embed` 提供前端静态资源
- 启动阶段执行数据库连通性检查和自动 migrate

## 文件变更

- 新增 `cmd/server` 启动入口
- 新增 `internal` 分层骨架
- 新增 `web` React 工程骨架
- 新增 `config/config.toml.example`

## 配置与依赖变更

- 新增 `POSTGRES_DSN`、`HTTP_ADDR`、`APP_ENV`、`LOG_LEVEL`
- 新增 `Gin`、`Fx`、`GORM`、`Viper`、`Zerolog`

## 测试结果

- 待执行 `go mod tidy` 和 `go test ./...`
- 前端依赖安装与 `vite build` 尚未执行

## 风险与后续建议

- 当前前端 `dist` 为占位页，待真实构建产物替换
- 下一步建议进入任务 `02` 或先补齐后端基础测试
