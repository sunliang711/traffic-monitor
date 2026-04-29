# 首次管理员随机密码初始化

## 任务背景

当 `INIT_ADMIN_USERNAME` 和 `INIT_ADMIN_PASSWORD` 都为空时，服务原先会跳过管理员初始化，空数据库场景下没有可登录账号。本次改为自动创建默认管理员，并输出一次性初始密码到启动日志。

## 实现方案

- 两个初始化配置都为空时，默认用户名使用 `admin`。
- 仅在目标管理员不存在且需要创建时，使用 `crypto/rand` 生成随机密码。
- 自动生成的密码只在创建成功后通过 `warn` 日志输出，日志提示首次登录后立即修改密码。
- 显式配置了 `INIT_ADMIN_PASSWORD` 时，不打印密码。
- 目标管理员已存在时，不重新生成密码、不重复创建账号。

## 文件变更

- `internal/service/auth_service.go`：新增默认管理员用户名、随机密码生成和初始化结果返回结构。
- `internal/bootstrap/admin_bootstrap.go`：根据初始化结果输出普通创建日志或自动密码日志。
- `internal/service/auth_service_test.go`：补充空配置自动创建、显式配置不泄露密码、已有默认管理员跳过的测试。
- `README.md`、`config/config.toml.example`：更新初始化账号密码说明。

## 配置与依赖变更

- 未新增环境变量。
- 未新增 Go 依赖。
- `INIT_ADMIN_USERNAME` 和 `INIT_ADMIN_PASSWORD` 仍必须同时配置或同时留空；同时留空时触发自动初始化。

## 测试结果

- `go test ./internal/service ./internal/bootstrap`：通过。
- `go test ./...`：通过。

## 风险与后续建议

- 自动生成的初始密码会出现在启动日志中，拥有日志访问权限的人可以看到该密码。
- 首次登录后应立即修改管理员密码。
