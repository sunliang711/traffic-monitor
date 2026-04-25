# 任务交付：machine-management

## 任务背景

根据 `task-04-machine-management`，实现远程机器管理、SSH Key 绑定和 SSH 连通性测试能力。

## 实现方案

- 新增 `machines` 表，保存主机、端口、用户、网卡、SSH Key 绑定和启用状态
- 新增机器 CRUD 接口
- 新增连接测试接口，验证 SSH 可连通且远端存在 `vnstat`
- 在现有 `APP_MASTER_KEY` 解密链路上增加私钥解密能力
- 增加 SSH 超时配置，控制连接测试时长

## 文件变更

- 新增 `internal/model/machine.go`
- 新增 `internal/repo/machine_repo.go`
- 新增 `internal/dto/machine.go`
- 新增 `internal/service/machine_service.go`
- 新增 `internal/service/machine_service_test.go`
- 新增 `internal/handler/machine_handler.go`
- 新增 `internal/bootstrap/ssh_client.go`
- 修改 `internal/bootstrap/crypto.go`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/config/*`
- 修改 `internal/handler/auth_handler.go`
- 修改 `config/config.toml.example`

## 配置与依赖变更

- 新增配置：
  - `ssh.dial_timeout`
  - `ssh.command_timeout`
- 新增环境变量：
  - `SSH_DIAL_TIMEOUT`
  - `SSH_COMMAND_TIMEOUT`
- 无新增第三方依赖

## 新增接口

- `GET /api/v1/machines`
- `POST /api/v1/machines`
- `GET /api/v1/machines/:id`
- `PATCH /api/v1/machines/:id`
- `DELETE /api/v1/machines/:id`
- `POST /api/v1/machines/:id/test-connection`

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go test ./...`
- 已新增 `internal/service/machine_service_test.go`

## 风险与后续建议

- 当前 SSH 连通性测试使用 `ssh.InsecureIgnoreHostKey()`，后续建议补主机指纹校验策略
- 当前删除 SSH Key 仍未禁止被机器引用时删除，建议尽快补保护
- 下一步建议进入任务 `05`，实现阈值配置模块
