# traffic-monitor 编码规范

本文档基于 Go 后端规则、安全规则和 API 规则裁剪整理，仅保留与 `traffic-monitor` 直接相关的约定。

## 1. 通用原则

- 严格遵循最小变更原则
- 只在明确需求范围内修改文件和代码
- 新增逻辑优先保证可读性和边界清晰，避免过度抽象
- 日志统一使用英文

## 2. 后端分层规范

- 目录建议：`cmd/`、`internal/`、`config/`、`web/`
- `internal` 下按职责拆分：`handler`、`service`、`repo`、`model`、`dto`、`middleware`、`scheduler`、`notifier`
- 强制分层：`Handler -> Service -> Repository -> Model/DTO`
- `Handler` 只做参数校验、响应封装、鉴权接入
- `Service` 负责业务逻辑、事务控制、调度编排
- `Repository` 只做数据访问，不包含业务判断
- 数据库实体和接口 DTO 严格分离

## 3. 技术选型约定

- HTTP 框架使用 `Gin`
- 依赖注入使用 `Uber Fx`
- 配置管理使用 `Viper`
- ORM 使用 `GORM`
- 日志使用 `Zerolog`
- 前端使用 `React + Vite`
- 非必要情况下不引入同类替代库

## 4. 命名规范

- 包名使用小写短词，不使用下划线
- 类型命名使用 `XxxHandler`、`XxxService`、`XxxRepo`、`XxxReq`、`XxxResp`
- 资源接口使用复数路径，如 `/api/v1/machines`
- JSON 字段统一 `snake_case`
- 数据库字段统一 `snake_case`
- ID 缩写统一大写，如 `SSHKeyID`、`MachineID`

## 5. API 规范

- 路由前缀统一为 `/api/v1`
- 所有接口使用统一响应结构

```json
{
  "code": 200,
  "data": {},
  "message": "success"
}
```

- 受保护接口必须经过统一鉴权中间件
- 所有外部输入必须校验
- 分页接口必须限制 `page_size`
- 错误响应禁止暴露内部实现细节

## 6. 配置规范

- 配置加载统一放在 `config` 包
- 默认加载顺序：默认值 < 配置文件 < 环境变量
- 新增配置项时必须补充默认值和校验
- 安全敏感配置只能通过受控配置注入
- 以下配置必须支持环境变量注入：
  - `POSTGRES_DSN`
  - `APP_MASTER_KEY`
  - `SESSION_SECRET`
  - `INIT_ADMIN_USERNAME`
  - `INIT_ADMIN_PASSWORD`

## 7. 数据库规范

- 使用 `GORM` 管理模型和 migrate
- 服务启动时自动执行 migrate
- 查询必须带 `context`
- 连接池参数从配置读取，禁止硬编码
- 样本表和告警表必须建立查询索引
- 告警去重键 `alert_key` 必须有唯一索引

## 8. 安全规范

- 管理员密码仅使用 `bcrypt` 存储
- SSH 私钥必须使用 `APP_MASTER_KEY` 加密后入库
- 主密钥不得写入配置文件、镜像和数据库
- 严禁记录私钥明文、Session 密钥、通知 Token 原文
- 禁止使用 `InsecureSkipVerify=true`
- 禁止拼接用户输入执行 `sh -c`
- 文件路径和外部输入必须做校验

## 9. 日志规范

- 使用结构化日志，禁止 `fmt.Println`
- 错误日志必须带 `err` 字段
- 采集日志至少包含：`machine_id`、`host`、`period_type`
- 通知日志至少包含：`alert_id`、`channel_type`、`success`
- 日志中禁止出现密码、私钥、完整 Bot Token

## 10. 并发与调度规范

- 共享状态必须加锁或通过 channel 控制
- 调度器和 worker pool 必须可优雅退出
- 单次 SSH 采集必须设置超时
- 重试次数和并发数必须从配置读取
- 禁止无限制启动 goroutine

## 11. 前端约定

- 页面按业务模块组织：认证、机器、阈值、通知、采集、告警
- API 调用统一封装，不在页面组件中散落请求细节
- 表单错误信息使用中文展示
- 登录态依赖 Cookie Session，不在前端存储 JWT
- 前端构建产物统一输出到供 `Go embed` 使用的目录

## 12. 测试约定

- 优先补充 Service 层单元测试
- SSH 执行器、通知发送器应通过接口隔离，便于 mock
- 解析 `vnstat` 输出应有表驱动测试
- 告警去重和阈值覆盖逻辑必须有单元测试
- 不在单元测试中连接真实远程机器
