# 任务交付：admin-password

## 任务背景

当前项目只支持首次启动时初始化管理员密码，管理员创建后无法通过页面或 API 修改密码。本次新增已登录管理员修改自身密码的能力。

## 实现方案

- 新增受登录保护的修改密码接口
- 修改密码时校验当前密码
- 新密码最小长度为 6 位
- 当前密码错误时返回明确提示
- 新密码使用 `bcrypt` 生成哈希后写入 `admins.password_hash`
- 修改成功后保留当前登录状态
- 前端账号菜单新增修改密码入口
- 登录失败时返回明确的用户名或密码错误提示

## 文件变更

- 修改 `internal/dto/auth.go`
- 修改 `internal/repo/admin_repo.go`
- 修改 `internal/service/auth_service.go`
- 修改 `internal/handler/auth_handler.go`
- 修改 `internal/service/auth_service_test.go`
- 修改 `web/src/App.tsx`
- 修改 `web/src/lib/app-types.ts`
- 修改 `web/src/lib/i18n.tsx`
- 修改 `web/src/styles.css`

## 配置与依赖变更

- 无新增配置项
- 无新增环境变量
- 无新增 Go 或前端依赖
- 无数据库表结构变更

## 新增接口

- `PATCH /api/v1/auth/password`

请求体：

```json
{
  "current_password": "old-password",
  "new_password": "new-password"
}
```

成功响应：

```json
{
  "code": 200,
  "data": null,
  "message": "success"
}
```

## 测试结果

- 已执行 `go test ./...`
- 已执行 `npm --prefix web run build`
- 已新增 `AuthService.ChangePassword` 成功、旧密码错误、新密码过短单元测试

## 风险与后续建议

- 当前接口只支持管理员修改自身密码，不支持重置其他管理员密码
- 当前密码修改成功后不强制退出已有 Session，适合当前单管理员自用场景
