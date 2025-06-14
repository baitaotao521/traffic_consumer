#!/bin/bash
# 脚本用于将 traffic_consumer.py 编译成 Linux 可执行文件
# 使用方法：在 WSL 中运行 bash compile_traffic_consumer.sh

set -e  # 遇到错误立即退出
echo "开始编译 traffic_consumer.py 为 Linux 可执行文件..."

# 创建工作目录
WORK_DIR="$HOME/traffic_consumer_build"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"
echo "工作目录: $WORK_DIR"

# 检查 traffic_consumer.py 是否存在
if [ ! -f "traffic_consumer.py" ]; then
    echo "正在复制 traffic_consumer.py 到工作目录..."
    # 假设脚本与 traffic_consumer.py 在同一目录
    cp "$(dirname "$0")/traffic_consumer.py" .
fi

# 创建并激活虚拟环境
echo "创建虚拟环境..."
python3 -m venv venv
source venv/bin/activate

# 安装依赖
echo "安装依赖..."
pip install --upgrade pip
pip install requests tqdm colorama apscheduler
pip install pyinstaller

# 编译 traffic_consumer.py
echo "使用 PyInstaller 编译 traffic_consumer.py..."
pyinstaller --onefile traffic_consumer.py

# 验证编译结果
if [ -f "dist/traffic_consumer" ]; then
    echo "编译成功! 可执行文件位于: $WORK_DIR/dist/traffic_consumer"
        
    # 添加执行权限
    chmod +x ~/traffic_consumer
    echo "已添加执行权限"
    
    echo "您可以通过以下命令运行程序:"
    echo "  ~/traffic_consumer"
else
    echo "编译失败，请检查错误信息"
    exit 1
fi

echo "编译过程完成" 