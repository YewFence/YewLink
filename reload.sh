#!/bin/bash
set -e

# 实际上只是简单的 docker compose down 和 docker compose up，但是会自动输出日志，方便查看配置生成是否成功。
# 以及本人确实懒得打这么多单词，所以就写了个脚本。
# 当然，你要是有神奇的控制面板直接重启 Agent 服务和重启 Init 服务也是一样的。

echo "=== YewLink 重载 ==="
echo ""
echo "这个脚本会："
echo "  1. 停止并移除所有容器"
echo "  2. 重新生成配置并启动 agent"
echo ""
echo "提示：容器的日志会直接输出，请留意配置生成是否成功。"
echo "      Ctrl+C 退出日志，容器会继续在后台运行。"
echo ""

read -p "确认重载？[y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  echo "已取消。"
  exit 0
fi

echo ""
echo ">>> 正在停止容器..."
docker compose down

echo ""
echo ">>> 正在重新启动..."
docker compose up -d

echo ""
echo ">>> 输出日志（Ctrl+C 退出，容器继续运行）..."
echo ""
docker compose logs -f
