<div align="center">
  <img src="https://raw.githubusercontent.com/YewFence/YewLink/main/img/logo.webp" alt="YewLink" height="120" />
</div>

<div align="center">
  <h1>YewLink</h1>
</div>

> 🚀 **YewLink** 是一款专为个人开发者与 Homelab 打造的零配置、无感知的基于 Infisical 的密钥同步与分发中枢。

本项目是 Infisical 官方 [Agent 模式](https://infisical.com/docs/integrations/platforms/infisical-agent) 的一个包装，也是一个开箱即用的本地服务 Secret 管理器。它通过提取云端凭证、本地落盘并使用 **软链接 (Symlink)** 的模式，为你提供了一个极其简单、透明且无供应商绑定的密钥管理方案。

---

## ✨ 核心特性

* 🤖 **服务自动发现**：彻底告别手写配置。只要云端有对应的服务文件夹，本地即可自动识别并同步，真正的即插即用。
* 🗂️ **Infisical 单项目复用**：巧妙利用 Infisical 的目录层级进行环境/设备隔离。仅需一个免费额度的 Project，就能轻松管理你所有的服务器节点和多个服务的密钥。
* 🔗 **极简注入与迁移**：生成的 `.env` 文件通过 `ln -s` 软链接一键注入业务容器。原有的部署逻辑零修改，极其适合存量服务的快速迁移。
* 🛡️ **透明审计，拒绝绑定**：密钥以普通 `.env` 文本落盘，对个人运维来说极其方便审查。即便未来更换密码管理器或停用本项目，已有服务仍能依靠本地的 `.env` 继续完美运行，**真正的零供应商绑定**。
* ⚡ **全自动初始化**：自带 Init 容器，自动拉取模板、生成配置并拉起官方 Agent 进行轮询，你只需提供基础认证信息即可。

---

## 💡 工作原理

1. **配置生成阶段 (Init)**: 内置的生成器读取你的认证凭证，自动扫描云端目录，为你生成 Infisical Agent 专属的轮询配置。
2. **同步守护阶段 (Daemon)**: Infisical Agent 守护进程在后台轮询 Infisical API，将更新的密钥写入对应的 `secrets/<服务名>.env` 文件。
3. **业务读取阶段 (Usage)**: 你的 Nginx、Postgres 等业务服务，只需要一个软链接指向这些 `.env` 文件，即可获得最新密钥。

---

## 🎯 典型应用场景

如果你习惯在机器上使用类似 `/opt/stacks/<service>/docker-compose.yml` 的方式来组织你的 Docker 服务，那么 YewLink 的角色就是一个 **为你的其他服务以中心化方式提供环境变量管理的「基础设施服务」**。

它在文件系统上的标准工作流通常如下：

```text
/opt/stacks/
├── YewLink/                  <-- 本项目，作为基础设施被优先启动
│   ├── docker-compose.yml
│   └── secrets/              <-- 自动拉取并分类生成各服务的 .env 文件
│       ├── nginx.env
│       └── vaultwarden.env
│
├── nginx/                    <-- 你的业务服务 A
│   ├── docker-compose.yml
│   └── .env -> ../YewLink/secrets/nginx.env  <-- 核心：通过软链接挂载
│
└── vaultwarden/              <-- 你的业务服务 B
    ├── docker-compose.yml
    └── .env -> ../YewLink/secrets/vaultwarden.env
```

**实际体验**：你只需要在 Infisical 云端更新密钥，YewLink 就会在后台将其同步至 `/path/to/YewLink/secrets/` 目录。当业务服务重新部署或重启时，它们会顺着这根软链接加载从云端同步的最新配置。

---

## 🚀 快速开始

### 1. 准备 Infisical 端配置

1. 登录 [Infisical](https://app.infisical.com)
> 或者你的私有部署实例，记得在 `config.yaml` 中修改 API 地址
2. 创建项目，记录 **项目 ID**
3. 为每个不同的环境/设备创建单独的文件夹（可选，用于环境隔离）
4. 为每个服务创建文件夹（如 `/vaultwarden`），在其中添加环境变量
5. 创建 Machine Identity（Universal Auth），记录 `Client ID` 和 `Client Secret`

#### 🛡️ Machine Identity 安全配置建议

| 配置项 | 建议值 | 说明 |
|--------|--------|------|
| 项目角色 | Viewer | Agent 只需读取 secrets，无需写入权限 |
| Client Secret Trusted IPs | 你服务器的公网 IP | 限制谁能用凭证换取 Token |
| Access Token Trusted IPs | 你服务器的公网 IP | 限制谁能用 Token 访问 API |
| Access Token TTL | 86400（1 天） | Token 有效期，过期后自动重新认证 |
| Access Token Max TTL | 86400（1 天） | Token 最大有效期上限 |
| Access Token Period | 0（留空） | 不使用可续期 Token，Agent 会自动重新认证 |

> **安全提示**：默认的 TTL 是 2592000 秒（ 30 天），建议缩短为 86400 秒（ 1 天）或者更短。即使 Token 泄露，1 天后就会失效。Agent 会自动处理 Token 续期，设短一点没有副作用。

### 2. 本地克隆与配置

```bash
# 1. 克隆到你的 docker 配置目录
cd /path/to/docker-configs
git clone https://github.com/yewfence/YewLink.git
cd YewLink

# 2. 创建认证文件 (请妥善保管)
echo "your-client-id" > client-id
echo "your-client-secret" > client-secret
chmod 600 client-secret

# 3. 编辑基础配置
cp config.example.yaml config.yaml
vim config.yaml # 填入 project_id 等基础信息，得益于自动发现，通常无需手动编写 services 列表
```

### 3. 启动 YewLink

> 具体被 Infisical CLI 读取的配置文件会由 init 服务自动生成

```bash
docker compose up -d && docker compose logs -f
```

查看日志，确认 Agent 成功启动并拉取了密钥

### 4. 注入密钥到业务服务

当 YewLink 启动并拉取密钥后，会在 `secrets/` 目录下生成对应的 `.env` 文件。
在你的业务服务目录下创建符号链接，将其指向生成的 secrets 文件：

```bash
cd /path/to/<service_name>

# 备份原有的 .env 文件（如果存在）
cp .env .env.backup

# 创建软链接
ln -sf ../YewLink/secrets/<service_name>.env .env
```

同时在业务的 `docker-compose.yml` 中添加 `env_file` 确保变量注入容器：

```yaml
services:
  <service_name>:
    env_file:
      - .env
    # ...其他配置
```

这样 secrets 会通过两种方式生效：
- **符号链接的 `.env`**：让 `docker-compose.yml` 中的 `${VAR}` 变量替换和默认值语法正常工作。
- **`env_file: .env`**：确保所有变量都直接注入到容器进程的运行环境中。

你可以参考 `yewlink-init` 容器的输出创建对应的符号链接。

输出示例：
```text
yewlink-init  | ✓ 自动发现 2 个服务: service1, services2
yewlink-init  | ✓ 已生成配置文件: /output/config.yaml
yewlink-init  |   - 项目 ID: xxxx
yewlink-init  |   - 环境: prod
yewlink-init  |   - 根文件夹: /
yewlink-init  |   - 服务数量: 2
yewlink-init  |     • service1
yewlink-init  |     • services2
yewlink-init  | 📋 在各服务目录下创建符号链接:
yewlink-init  |     cd ../service1 && ln -sf ../YewLink/secrets/service1.env .env
yewlink-init  |     cd ../services2 && ln -sf ../YewLink/secrets/services2.env .env
yewlink-init  | 💡 建议先备份原 .env 文件（如果有）
yewlink-init  |     cd ../service1 && mv .env .env.bak
yewlink-init  |     cd ../services2 && mv .env .env.bak
```

---

## 🛠️ 添加新服务与迁移

1. **Infisical 云端**：创建文件夹 `/<服务名>`，录入环境变量。*(开启了自动发现功能后可以跳过修改 config.yaml 的步骤)*
2. **YewLink 重载**：执行 `./reload.sh` 触发自动发现并重新生成配置。

> 如果创建了新服务需要手动执行重载，以重新生成模板文件（实际上 `docker compose up yewlink-init` 即可，当然完全重新构建容器也没问题）
> 如果只是修改了某个服务的环境变量，YewLink 会在下一个轮询周期自动更新对应的 `.env` 文件，可以等待它自动更新，无需手动重载，或者 `docker compose restart yewlink` 服务使其立即更新。
> 需要注意，Infisical CLI 容器偶尔会在重启后认证失败，这时可以通过 `docker compose up yewlink --force-recreate` 令它重新认证即可解决问题

3. **业务端链接**：如上文所述，使用 `ln -sf ../YewLink/secrets/<服务名>.env .env` 创建软链接。并在 `docker-compose.yml` 中添加 `env_file: .env` 即可。

💡 **从已有服务平滑迁移**：
将原 `.env` 文件中的变量复制到 Infisical 网页端并导入，然后按照上述步骤用软链接替换掉原来的 `.env` 即可完成迁移。

---

### 📂 部署时目录结构

```text
YewLink/
├── docker-compose.yml    # YewLink 的 Docker Compose 配置
├── config.yaml           # 基础配置文件（填写 Project ID 等）
├── config.example.yaml   # 配置示例
├── client-id             # Machine Identity ID
├── client-secret         # Machine Identity Secret
├── reload.sh             # 快速重载脚本
└── secrets/              # 生成的 secrets 文件（自动创建，请注意安全防护）
    ├── nginx.env
    ├── postgres.env
    └── ...
```

## 补充说明

### 关于 `docker-compose.yml` 中使用的镜像版本 (官方 Bug 修复)

本项目默认镜像使用 `ghcr.io/yewfence/yewlink-init` 与 `ghcr.io/yewfence/infisical-cli:latest`。
由于[官方 CLI](https://github.com/infisical/cli) 存在 BUG，`agent` 模式下无法输出变量的注释到 `.env` 文件中（详见 [ISSUE #103](https://github.com/Infisical/cli/issues/103)）。我已经提交了修复 [PR #104](https://github.com/Infisical/cli/pull/104)，在合并前，我自行构建了修复版本的镜像并跟随官方更新。

如果你不介意该问题，可以修改 `docker-compose.yml` 替换为官方镜像 `infisical/cli:latest`。

### 本地开发与测试

修改了生成器或模板后，可以用 `--build` 在本地构建 init 容器镜像进行测试：

```bash
docker compose up -d --build
```

如果需要在本地直接运行生成器（不通过 Docker），可以手动编译：

```bash
cd generator
go build -ldflags="-s -w" -o yewlink-init .
./yewlink-init
```

也可以直接从 [Release 页面](https://github.com/YewFence/YewLink/releases/latest) 下载预编译的二进制文件。

### 注意事项与安全防御

- `client-secret` 文件权限建议设置为 `600`。
- 启动顺序：先启动 YewLink，等 secrets 文件生成后再启动其他服务。
- Agent 默认每 5 分钟轮询一次更新。
- 项目已配置 [lefthook](https://github.com/evilmartians/lefthook) + [gitleaks](https://github.com/gitleaks/gitleaks)，每次提交前自动扫描暂存区，防止意外提交密钥。若开发本项目，建议首次克隆后运行 `lefthook install` 以确保密钥不会被错误提交。

---


## 🤔 诞生背景

**“为什么我要做这个项目？”**

起初，我习惯使用 `docker compose` 来组织服务器上的所有应用。但我很快发现了一个严重的问题： **所有关键的密钥（Secret）、数据库密码和 API Key 全都散落、硬编码在各个服务文件夹的 `.env` 文件里**。

这些配置文件不仅 **缺乏统一备份** ，而且变成了一个个“信息孤岛”。一旦服务器损坏或迁移，如果忘记备份那些分散的 `.env` 文件，将会是一场灾难。为了解决这个问题，我迫切需要一个能把所有密钥都统一收拢在云端、实现 **中心化管理** 的方案。但同时，我又**绝对不想被某一家密码管理服务商绑定**——如果哪天云服务挂了、涨价了，或者我想换一家，我的本地服务必须依然能完美运行。

于是，**YewLink** 诞生了。它启发于 Infisical 官方的 Agent 模式：监听云端 Secret 变更并同步到本地，但我对它进行了一定的改造和增强，使其成为一个方便使用、无感知的密钥同步中枢。

---

## ⚠️ 安全须知与最佳实践

### 关于 Secret 落盘的安全权衡
本项目采用了将加密环境变量在本地解密并以普通 `.env` 文本文件落盘的同步机制。从绝对的“零信任安全”角度来看，这种方法**不如**企业级方案（例如原生 Kubernetes Secrets 或将凭证通过专用 SDK 直接注入应用程序内存）安全。一旦攻击者拿到了你服务器的读取权限，他们就可以直接读取这些 `.env` 纯文本文件。

但对于我而言，这是一个经过我思考的**妥协与取舍**：

*  ✅ **绝对掌控力**：随时 `cat` 检查密钥是否正确分发，排障极快。
*  ✅ **真正的零绑定**：一旦你决定移除 YewLink，系统只需依靠本地留存的 `.env` 即可继续完美运转，彻底告别被某家工具绑架的烦恼。
*  风险一致：在引入任何其他第三方密码管理器之前，你的密钥也必须以某种形式落盘（无论是 `.env`、数据库还是专用的 Secret Store）。相比之下，YewLink 只是把这个过程透明化了，并没有引入额外的风险。

请自行判断安全风险并选择是否使用。

### 推荐使用官方 Infisical Cloud
YewLink 完美兼容你自行部署的私有 Infisical 实例（只需在 `config.yaml` 中配置相应的 API 地址即可）。

**但是，我们极其推荐你直接使用免费的 [Infisical Cloud](https://app.infisical.com/) 官方云服务**：
1.  **充分利用免费额度**：官方免费版限制创建 3 个 Project。但得益于 YewLink 的多目录环境复用特性，**你只需占用 1 个免费项目额度**，利用文件夹层级，就能管理你所有设备和无数个服务的密钥
2.  **享受官方企业级 SLA**：将最核心的“根金库”交由官方维护，拥有极高的可用性和数据灾备保证，免去了自己搭建、备份高可用数据库、升级系统版本等一系列无底洞般的运维烦恼。
