# 使用轻量的Python 3.12 slim镜像
FROM python:3.12-slim

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY requirements.txt .

# 安装依赖
RUN pip install --no-cache-dir -r requirements.txt

# 复制所有项目文件到工作目录
COPY . .

# 暴露Web UI端口
EXPOSE 5001

# 默认启动命令
# 运行 traffic_consumer.py，它会默认启动Web UI
CMD ["python", "traffic_consumer.py"]