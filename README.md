# Infisical Agent 容器配置生成器

一个用于生成 Infisical Agent 配置文件的小工具，配合 Docker 部署实现 secrets 的统一管理。

## 快速开始

### 1. 准备 Infisical

1. 登录 [Infisical](https://app.infisical.com)
2. 创建项目，记录 **项目 ID**
3. 为每个不同的环境创建单独的文件夹（可选，可以用于区分不同设备等）
4. 为每个服务创建文件夹（如 `/vaultwarden`），添加环境变量
5. 创建 Machine Identity（Universal Auth），记录 `Client ID` 和 `Client Secret`

#### Machine Identity 配置建议

| 配置项 | 建议值 | 说明 |
|--------|--------|------|
| 项目角色 | Viewer | Agent 只需读取 secrets，无需写入权限 |
| Client Secret Trusted IPs | 你服务器的公网 IP | 限制谁能用凭证换取 Token |
| Access Token Trusted IPs | 你服务器的公网 IP | 限制谁能用 Token 访问 API |
| Access Token TTL | 86400（1 天） | Token 有效期，过期后自动重新认证 |
| Access Token Max TTL | 86400（1 天） | Token 最大有效期上限 |
| Access Token Period | 0（留空） | 不使用可续期 Token，Agent 会自动重新认证 |

> **安全提示**：默认的 TTL 是 2592000 秒（ 30 天），建议缩短为 86400 秒（ 1 天）或者更短。即使 Token 泄露，1 天后就会失效。Agent 会自动处理 Token 续期，设短一点没有副作用。

### 2. 克隆并配置

```bash
# 克隆到你的 docker 配置目录
cd /path/to/docker-config
git clone https://github.com/yewfence/infisical-agent.git infisical-agent
cd infisical-agent

# 创建认证文件
echo "your-client-id" > client-id
echo "your-client-secret" > client-secret
chmod 600 client-secret

# 编辑服务配置
cp config.example.yaml config.yaml
vim config.yaml
```

### 3. 启动 Agent

> 具体被 Infisical CLI 读取的配置文件会由 init 容器自动生成，无需手动下载或运行生成器。

```bash
docker compose up -d
```

### 4. 生成 .env 软链接
> 具体命令可以参考配置生成器的输出，此处命令仅作示例
```bash
📋 在各服务目录下创建符号链接:
    cd ../nginx && ln -sf ../infisical-agent/secrets/nginx.env .env

📋 同时在 docker-compose.yml 中添加 env_file:
    env_file: .env

💡 建议先备份原 .env 文件
    mv ../nginx/.env ../nginx/.env.bak
```


## 配置说明

### config.yaml

```yaml
# Infisical 项目 ID
project_id: "your-project-id"

# 环境 (dev/staging/prod)
environment: "prod"

# 轮询间隔
polling_interval: "300s"

# 可选：读取配置的根文件夹，可以用来在一个项目中区分不同环境
# root_folder: "/project-a"

# 服务列表 - 每个服务名称对应 Infisical 中读取配置的根文件夹下的一个文件夹
services:
  - nginx
  # - vaultwarden
  # - postgres
```

### 目录结构

```
infisical-agent/
├── docker-compose.yml    # Agent 容器配置
├── config.yaml           # 服务列表（需自行创建并编辑）
├── config.example.yaml   # 配置示例
├── client-id             # Machine Identity ID（需自行创建）
├── client-secret         # Machine Identity Secret（需自行创建）
└── secrets/              # 生成的 secrets 文件（自动创建，请注意安全问题）
    ├── vaultwarden.env
    ├── postgres.env
    └── ...
```

> 配置模板 `config.yaml.tmpl` 和生成器已烤入 init 容器镜像

## 在其他服务中使用

在业务服务目录下创建符号链接，将 `.env` 指向 Infisical 生成的 secrets 文件：

```bash
# 以 vaultwarden 为例
cd /path/to/vaultwarden

# 备份原有的 .env 文件（如果存在）
cp .env .env.backup

# 创建符号链接
ln -sf ../infisical-agent/secrets/vaultwarden.env .env
```

同时在 `docker-compose.yml` 中添加 `env_file` 确保变量注入容器：

```yaml
services:
  vaultwarden:
    env_file:
      - .env
    # ...其他配置
```

这样 secrets 会通过两种方式生效：
- **符号链接的 `.env`**：让 `${VAR}` 变量替换和默认值语法正常工作
- **`env_file: .env`**：确保所有变量都注入到容器环境中

## 添加新服务

1. **Infisical**：创建文件夹 `/<服务名>`，添加环境变量
2. **config.yaml**：在 `services` 列表中添加服务名
3. **重启容器**：`docker compose up -d --force-recreate`
4. **业务服务**：
   - 创建符号链接：`ln -sf ../infisical-agent/secrets/<服务名>.env .env`
   - 在 `docker-compose.yml` 中添加 `env_file: .env`

## 自行编译

如果需要在本地使用生成器（不通过 Docker），可以手动编译：

```bash
cd generator
# infisical-config-generator
go build -ldflags="-s -w" -o icg .
./icg
```

也可以直接从 [Release 页面](https://github.com/YewFence/infisical-agent/releases/latest) 下载预编译的二进制文件。

## 注意事项

- `client-secret` 文件权限建议设置为 `600`
- 启动顺序：先启动 infisical-agent，等 secrets 文件生成后再启动其他服务
- Agent 默认每 5 分钟轮询一次更新
- 项目已配置 [lefthook](https://github.com/evilmartians/lefthook) + [gitleaks](https://github.com/gitleaks/gitleaks)，每次提交前自动扫描暂存区，防止意外提交密钥。首次克隆后运行 `lefthook install` 即可启用

## 从已有服务迁移
可以参考[迁移说明](./INFISICAL-MIGRATION.md)
