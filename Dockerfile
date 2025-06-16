FROM python:3.11-alpine

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && \
    rm -rf /root/.cache/pip

# 复制程序文件
COPY traffic_consumer.py .

# 创建配置目录
RUN mkdir -p /root/.traffic_consumer

# 设置容器启动命令
ENTRYPOINT ["python", "traffic_consumer.py"]

# 默认参数，可在运行时覆盖
CMD ["--threads", "8"] 