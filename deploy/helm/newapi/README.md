# new-api Helm Chart

This chart deploys new-api with built-in PostgreSQL and Redis by default.
It is preconfigured for the CMDB-auth integration build.

## Quick start

```bash
helm install new-api ./deploy/helm/new-api-0.1.0.tgz
```

The default values deploy:

- new-api application from the local/custom `new-api:latest` image
- PostgreSQL 15
- Redis 7
- PVCs for `/data` and `/app/logs`
- CMDB JWT authentication enabled
- CMDB access token read from the `Access-Token` header or `token` cookie
- A separate `newapi_session` cookie to avoid conflicts with CMDB's `session` cookie

## External PostgreSQL and Redis

```yaml
secret:
  sessionSecret: "replace-with-a-long-random-secret"
  sqlDsn: "postgresql://user:password@postgres.example:5432/new-api?sslmode=disable"
  redisConnString: "redis://:password@redis.example:6379/0"

database:
  type: postgresql
  postgresql:
    builtIn: false

redis:
  enabled: true
  builtIn: false
```

## SQLite mode

SQLite is useful only for small single-pod deployments:

```yaml
database:
  type: sqlite

redis:
  enabled: false
```

## Existing Secret

Set `secret.existingSecret` to a Secret containing these keys:

- `SESSION_SECRET`
- `SQL_DSN`, unless using SQLite
- `REDIS_CONN_STRING`, when Redis is enabled

Optional keys:

- `CRYPTO_SECRET`
- `CMDB_JWT_SECRET`, when `CMDB_AUTH_ENABLED=true`
- `POSTGRES_PASSWORD`, when using built-in PostgreSQL
- `MYSQL_ROOT_PASSWORD`, when using built-in MySQL
- `REDIS_PASSWORD`, when using built-in Redis with a password

## CMDB authentication

To let new-api trust CMDB `Access-Token` JWTs while keeping quota, groups, API tokens, model permissions, and audit data in new-api:

```yaml
env:
  CMDB_AUTH_ENABLED: "true"
  CMDB_AUTH_HEADER: Access-Token
  CMDB_AUTH_COOKIE: token
  CMDB_AUTH_ALLOW_AUTHORIZATION: "true"
  CMDB_AUTH_MATCH_USERNAME: "false"
  SESSION_COOKIE_NAME: newapi_session

secret:
  cmdbJwtSecret: "change-me"
```

These are the chart defaults. With this first-stage integration, the CMDB user must already exist in new-api with the same email. User auto-sync can be added later.
