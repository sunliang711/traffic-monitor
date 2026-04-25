# 任务交付：ssh-key-management

## 任务背景

根据 `task-03-ssh-key-management`，实现 SSH 私钥导入、系统生成 keypair、私钥加密存储和 SSH Key 管理接口。

## 实现方案

- 新增 `ssh_keys` 表，保存名称、来源类型、密钥类型、公钥、指纹和私钥密文
- 使用 `APP_MASTER_KEY` 作为主密钥，采用 `AES-256-GCM` 加密私钥
- `generate` 固定生成 `ed25519` keypair
- `import` 兼容导入多种常见私钥类型，并自动提取公钥与指纹
- API 仅返回可安全暴露的字段，不返回私钥明文和私钥密文

## 文件变更

- 新增 `internal/bootstrap/crypto.go`
- 新增 `internal/model/ssh_key.go`
- 新增 `internal/repo/ssh_key_repo.go`
- 新增 `internal/service/ssh_key_service.go`
- 新增 `internal/service/ssh_key_service_test.go`
- 新增 `internal/handler/ssh_key_handler.go`
- 新增 `internal/dto/ssh_key.go`
- 修改 `internal/config/*`
- 修改 `internal/bootstrap/database.go`
- 修改 `internal/app/module.go`
- 修改 `internal/handler/auth_handler.go`
- 修改 `config/config.toml.example`

## 配置与依赖变更

- 新增配置：`security.app_master_key`
- 新增环境变量：`APP_MASTER_KEY`
- `APP_MASTER_KEY` 要求为 base64 编码后的 32 字节密钥
- 无新增第三方加密库，SSH 继续使用 `golang.org/x/crypto/ssh`

## 新增接口

- `GET /api/v1/ssh-keys`
- `POST /api/v1/ssh-keys/import`
- `POST /api/v1/ssh-keys/generate`
- `GET /api/v1/ssh-keys/:id/public-key`
- `DELETE /api/v1/ssh-keys/:id`

## 测试结果

- 已执行 `gofmt -w cmd internal`
- 已执行 `go mod tidy`
- 已执行 `go test ./...`
- 已新增 `internal/service/ssh_key_service_test.go`

## 风险与后续建议

- 当前删除 SSH Key 未做引用保护，建议在机器管理任务里增加“被机器引用时禁止删除”
- 当前只实现加密存储，不支持主密钥轮换
- 下一步建议进入任务 `04`，把 SSH Key 和机器管理串起来
