# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Infisical Agent 配置生成器 — 一个 Go CLI 工具，从简化的 `config.yaml` 生成 Infisical CLI Agent 所需的完整配置文件，配合 Docker 部署实现多服务 secrets 统一管理。

## 常用命令

### Docker 部署（主要使用方式）
```bash
docker compose up -d              # 启动（使用远程镜像）
docker compose up -d --build      # 本地构建 init 容器并启动（修改生成器/模板后测试用）
docker compose up -d --force-recreate  # 强制重建（配置变更后）
```

### 本地构建生成器
```bash
cd generator && go build -ldflags="-s -w" -o icg .
./icg  # 使用默认参数运行
./icg -services config.yaml -template config.yaml.tmpl -output config-no-manually-edit.yaml
```

### Git Hooks
```bash
lefthook install  # 首次克隆后启用 pre-commit hook（gitleaks 密钥扫描）
```

## 架构

项目采用两阶段 Docker 容器模式：

1. **config-init 容器**（一次性）：运行 Go 生成器 `icg`，读取用户的 `config.yaml` + 内置模板 `config.yaml.tmpl`，输出 Infisical CLI 能理解的完整配置到共享 volume
2. **infisical-agent 容器**（常驻）：使用官方 `infisical/cli` 镜像，以 agent 模式运行，按配置轮询 Infisical API 并将 secrets 写入 `secrets/<服务名>.env`

### 核心文件

- `generator/main.go` — 单文件 Go 程序，包含配置读取、校验、模板渲染全部逻辑。输出的 CLI 名称为 `icg`
- `config.yaml.tmpl` — Go text/template 模板，渲染为 Infisical Agent 配置。注意模板中使用双层花括号 `{{"{{"}}` 来转义 Infisical 自身的模板语法
- `config.example.yaml` — 用户配置示例，字段包括 `project_id`、`environment`、`polling_interval`、`root_folder`（可选）、`services` 列表
- `Dockerfile` — 多阶段构建：Go builder → Alpine 运行时，入口点直接调用 icg

### 数据流
```
config.yaml (用户编辑) + config.yaml.tmpl (内置模板)
  → icg 生成器渲染 → Infisical Agent 配置
  → infisical/cli agent 读取 → 轮询 Infisical API
  → secrets/*.env (自动生成)
  → 业务服务通过符号链接读取 .env
```

## CI/CD

- `build.yml` — main 分支 generator/ 变更时构建（linux/windows amd64）
- `release.yml` — 推送 `v*` tag 时构建二进制并创建 GitHub Release
- `docker.yml` — 推送 `v*` tag 时构建并推送 Docker 镜像到 `ghcr.io/yewfence/infisical-config-init`

## 注意事项

- `client-id`、`client-secret`、`config.yaml`、`secrets/` 均在 `.gitignore` 中，绝对不要提交
- Go 模块路径为 `github.com/yewyard/infisical-config-generator`，依赖仅 `gopkg.in/yaml.v3`
- Dockerfile 使用 Go 1.21，CI 使用 Go 1.23
