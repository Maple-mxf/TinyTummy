# 使用官方Go镜像作为构建基础
FROM golang:1.21

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置工作目录
WORKDIR /app

# 复制go代码和模板文件
COPY . .

# 下载依赖并构建
RUN go mod tidy
RUN go build -o baby-feeding-tracker main.go

# 暴露服务端口
EXPOSE 8080

# 设置运行时环境变量（可在docker run时覆盖）
ENV MYSQL_DSN="root:password@tcp(localhost:3306)/feeding?parseTime=true"

# 运行程序
CMD ["./baby-feeding-tracker"]