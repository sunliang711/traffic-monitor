# 任务交付：react-console

## 任务背景

根据 `task-09-react-console`，实现 MVP 所需的 React 管理后台，覆盖登录、机器管理、SSH Key、阈值、通知、采集样本和告警记录。

## 实现方案

- 在不新增前端依赖的前提下，使用单页 React 结构实现管理台
- 使用统一 `api.ts` 封装接口请求，并带上 Cookie Session
- 页面按模块拆分为：
  - 登录
  - 总览
  - 机器管理
  - SSH Key 管理
  - 阈值配置
  - 通知渠道配置
  - 样本查看
  - 告警查看
- 样式使用纯 CSS，保持轻量并适配桌面与移动端

## 文件变更

- 新增 `web/src/api.ts`
- 新增 `web/src/types.ts`
- 重写 `web/src/App.tsx`
- 重写 `web/src/styles.css`

## 完成情况

- React 源码已完成
- 已按现有后端接口完成页面映射和表单提交流程
- 已兼容 Cookie Session 登录态
- 已支持总览卡片、机器 CRUD、SSH Key 导入/生成、阈值编辑、通知配置、样本查看和告警查看
- 已执行 `npm install`
- 已执行 `npm run build`
- `web/dist` 已刷新为新的前端构建产物

## 后续建议

1. 通过后端或 Docker 启动服务验证内嵌页面
2. 再进入 Docker 交付和联调阶段
