# new-api 接入 cmdb 统一鉴权方案

## 背景

new-api 当前有两套认证入口：

- 仪表盘 API：Gin session，登录后 session 内保存 `id`、`username`、`role`、`status`、`group`，`middleware.UserAuth/AdminAuth/RootAuth` 依赖这些字段。
- OpenAI 兼容调用：new-api 自己的 API token，`TokenAuth` 按 token 找到 new-api 本地用户、额度、分组和模型权限。

cmdb 当前登录后签发 HS256 JWT，前端放在 `Access-Token` 请求头；后端用 `SECRET_KEY` 解 token，从 payload 的 `sub` 邮箱查 cmdb 用户，随后重建 Flask session 和 ACL 上下文。

因此推荐把 cmdb 作为身份源，new-api 保留本地用户表用于额度、分组、令牌、日志、订阅等业务数据。不要直接让 new-api 读写 cmdb 用户表作为自己的用户表，否则会把计费、删除、状态、角色字段耦合到 cmdb 的 ACL 模型里，后续升级和回滚都更危险。

## 推荐方案：cmdb JWT 桥接 + 本地用户映射

### 登录与访问链路

1. 用户先登录集成平台/cmdb，前端拿到 cmdb 的 `Access-Token`。
2. 嵌入 new-api 时，通过同域反向代理、iframe 初始化参数或前端网关注入，把 cmdb token 传给 new-api。
3. new-api 增加 `CMDBAuth` 中间件：
   - 优先识别 `Access-Token` 或 `Authorization: Bearer <cmdb-jwt>`。
   - 用配置的 `CMDB_JWT_SECRET` 校验 HS256、`exp`、`iat`。
   - 从 `sub` 取邮箱，必要时调用 cmdb `/v1/acl/users/info` 或新增 `/v1/acl/users/me` 获取 `uid/username/nickname/email/role/parents/block`。
   - 按 `cmdb_uid` 或 `email` 映射到 new-api 本地用户；不存在则自动创建。
   - 把映射后的 new-api 用户写入 Gin context/session：`id/username/role/status/group`。
4. new-api 原有 `UserAuth/AdminAuth/RootAuth` 继续基于本地角色工作；模型分组、额度、token 仍走 new-api 原逻辑。

### 用户映射字段

建议在 new-api `users` 增加字段：

- `external_provider`：固定为 `cmdb`。
- `external_id`：cmdb `uid`，唯一索引建议为 `(external_provider, external_id)`。
- `email`：同步 cmdb 邮箱，用于兜底匹配。
- `display_name`：cmdb nickname/name。
- `status`：cmdb block=true 映射为 disabled。

如果暂时不改表，也可以先用 `email` 做匹配，但长期建议引入 `external_id`，避免邮箱变更导致新建重复账号。

### 角色与权限映射

推荐做成配置，不把 cmdb 角色名写死：

```text
CMDB_ROLE_ROOT=平台超级管理员,ops_admin
CMDB_ROLE_ADMIN=cmdb_admin,new_api_admin
CMDB_DEFAULT_GROUP=default
CMDB_AUTO_CREATE_USER=true
```

映射策略：

- cmdb 超级管理员或指定父角色 -> new-api root。
- cmdb 应用管理员或指定角色 -> new-api admin。
- 其他未封禁用户 -> new-api common。
- 额度、可用模型、分组仍由 new-api 管理，可在首次创建时给默认额度和默认 group。

### 安全边界

- cmdb token 只用于登录 new-api 仪表盘，不替代 new-api API token；AI 调用继续使用 new-api 令牌，才能保留额度扣费、模型限制和审计。
- iframe 集成建议优先同域反向代理，例如 `/cmdb/` 与 `/new-api/` 共站点，降低第三方 Cookie/SameSite 问题。
- 如果跨域，建议增加一个后端 SSO 交换接口：集成平台拿 cmdb token 调 new-api `/api/sso/cmdb/exchange`，new-api 校验后设置自己的 session cookie。
- JWT 密钥必须通过 Secret 注入，不要写入 ConfigMap。
- new-api 应该支持 token revoke 检查：如果 cmdb 使用 Redis denylist，new-api 可以调用 cmdb introspection/me 接口确认 token 未注销。

## 最小改造清单

1. new-api 新增 `service/cmdb_auth.go`：校验 cmdb JWT、可选拉取 cmdb 用户信息、返回已映射的 new-api 本地用户。
2. new-api 在 `middleware/auth.go` 的 session 不存在时尝试 cmdb token，并填充 Gin context/session。
3. new-api 用户模型后续增加外部身份字段和跨库兼容迁移。
4. new-api 新增配置项：
   - `CMDB_AUTH_ENABLED`
   - `CMDB_JWT_SECRET`
   - `CMDB_BASE_URL`
   - `CMDB_TOKEN_HEADER`
   - `CMDB_AUTO_CREATE_USER`
   - `CMDB_DEFAULT_GROUP`
   - `CMDB_ROLE_ROOT`
   - `CMDB_ROLE_ADMIN`
5. 前端嵌入时复用 cmdb 的 `Access-Token`，或走后端 SSO exchange 后只依赖 new-api session。
6. 禁用或隐藏 new-api 密码登录/注册入口，避免双身份入口造成用户混乱。

## 分阶段落地

第一阶段：只做仪表盘 SSO。cmdb 用户能进入 new-api，new-api 自动创建/映射用户，API token 与额度仍在 new-api 内管理。

第二阶段：把 cmdb ACL 角色映射到 new-api 管理权限，并提供后台“外部用户同步”页面或命令。

第三阶段：可选做 cmdb 应用入口深度集成，包括统一菜单、单点退出、审计日志回传 cmdb。

## 当前已实现的第一步

当前代码已支持“只做认证打通，不自动创建用户”：

- `CMDB_AUTH_ENABLED=true` 开启 cmdb JWT 鉴权。
- `CMDB_JWT_SECRET` 使用 cmdb 的 `SECRET_KEY`。
- 默认读取请求头 `Access-Token`，也兼容 `Authorization: Bearer <jwt>`。
- 如果请求头没有 token，默认从 Cookie `token` 读取 cmdb JWT，可通过 `CMDB_AUTH_COOKIE` 修改。
- new-api 前端支持从 URL 参数 `cmdb_access_token` 读取 cmdb token，保存为本域 `Access-Token` 后清理 URL。
- 默认使用 cmdb JWT 的 `sub`/`email` 匹配 new-api 本地用户 `email`。
- 如果设置 `CMDB_AUTH_MATCH_USERNAME=true`，也允许用 cmdb JWT 的 `username` 匹配 new-api 本地用户名。
- 如果设置 `CMDB_AUTH_USERINFO_URL`，new-api 会携带 token 请求该 URL，成功后使用返回的 `email/username` 进行匹配；这个接口也可用于让 cmdb 的 token revoke/denylist 立即生效。

示例：

```bash
CMDB_AUTH_ENABLED=true
CMDB_JWT_SECRET=<cmdb SECRET_KEY>
CMDB_AUTH_USERINFO_URL=http://cmdb-api.default.svc.cluster.local:5000/v1/acl/users/info
```

未同步到 new-api 的 cmdb 用户会被拒绝登录；等用户同步策略确定后，再把“找不到本地用户”的分支改为自动创建或按 `external_provider/external_id` 绑定。

## addc-ui 集成入口

兄弟项目 `addc-ui` 已将左侧菜单的 `Model APIs` 从新窗口跳转改成平台内路由 `/models-api`：

- `/models-api` 使用 iframe 承载 new-api。
- iframe 地址来自 `VITE_MODEL_API_URL`，未配置时默认 `http://api.mass.netnic.cn:8080/`。
- addc-ui 会把当前登录 token 以 `cmdb_access_token` 参数传给 new-api。
- new-api 前端读取该参数并保存，后续 API 请求携带 `Access-Token`，进入后端 cmdb JWT 鉴权链路。

生产环境建议把 new-api 配成同站点路径或可信子域，并在反向代理层避免记录带有 `cmdb_access_token` 的查询串日志。

## 需要确认的点

- 集成平台嵌入 new-api 是同域路径、子域名，还是跨域 iframe。
- cmdb 是否能新增一个稳定的 `/me` 或 token introspection 接口，返回 `uid/username/email/nickname/block/roles`。
- new-api 的管理员是否全部来自 cmdb，还是仍保留本地 root 作为应急入口。
- 用户首次进入 new-api 时默认额度、默认 group、默认模型权限如何分配。
