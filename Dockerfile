# 第一阶段：构建阶段
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制项目文件到容器中（排除 .dockerignore 中的内容）
COPY . .

# 构建应用并清理缓存
RUN go build -o /app/main . && \
    rm -rf /go/pkg/mod /root/.cache/go-build

# 第二阶段：运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制构建好的二进制文件
COPY --from=builder /app/main /app/main

# 设置环境变量
ENV ROOT=.
ENV TMP_DIR=tmp

# 运行应用
CMD ["./main", "server"]
