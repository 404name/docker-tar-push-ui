# 镜像UI上传工具 

image-upload-portal 解决部分场景无法使用命令行上传镜像的问题

![](./docs/image-upload-portal.jpg)

# install

**运行项目**
- linux: ./image-upload-portal server --port=8088
- windows: ./image-upload-portal.exe server --port=8088
- docker: docker run -d --name image-upload-portal -p 8088:8088 registry.cn-hangzhou.aliyuncs.com/404name/image-upload-portal:latest


**本地开发**
go mod tidy
go run ./ server

docker build -t image-upload-portal:latest  .
docker run -d --name image-upload-portal -p 8088:8088 image-upload-portal:latest


**实现热加载**
go get -u github.com/cosmtrek/air
air init
air


## 功能

- 支持上传harbor / 阿里云
- 支持UI模式和命令行两种模式

# TODO

- [x] dockerfile打包
- [ ] 流水线自动打包linux+windows+docker仓库 + 自动发布tag
- [ ] YAML适配
- [x] 支持阿里云推送
- [ ] 支持华为云和dockerhub等标准场景（网络限制，测试不了dockerhub）
- [ ] 支持默认追加https://
- [ ] 支持上传成功后提示 

华为云的scope="repository:name404/alist:" 似乎需要适配下
```
2025/01/08 13:02:05 push.go:243: [INFO] Received Www-Authenticate header: Bearer realm="https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/",service="dockyard",scope="repository:name404/alist:"
2025/01/08 13:02:05 auth.go:107: [INFO] Parsed Www-Authenticate header - realm: https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/, service: dockyard, scope: repository:name404/alist:,push
2025/01/08 13:02:05 auth.go:22: [INFO] Parsed Www-Authenticate header - realm: https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/, service: dockyard, scope: repository:name404/alist:,push
2025/01/08 13:02:05 auth.go:31: [INFO] Constructed token request data: map[password:[355d42dfa6a82ad2ea20f69d4f846522f59b3a4d116b2180115c5a789dea9cf9] scope:[repository:name404/alist:,push] service:[dockyard] username:[cn-north-4@9OU2ASO8F9VIDXZAXKC6]]
2025/01/08 13:02:05 auth.go:44: [INFO] Sending token request to https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/
2025/01/08 13:02:05 auth.go:53: [INFO] Received token response with status code: 404
2025/01/08 13:02:05 auth.go:57: [ERROR] Token request failed with status code: 404

```

# 说明

- 底层GO推送镜像代码来自：https://github.com/silenceper/docker-tar-push
- 基于这个实现了指定项目组 + 支持阿里云仓库认证