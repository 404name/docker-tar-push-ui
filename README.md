# 一、镜像UI上传工具 

> - 背景：[open：在Harbor UI界面实现镜像上传/下载](https://github.com/goharbor/harbor/issues/17028) + [由监管要求下架 dockerhub 镜像](https://www.geekery.cn/free-service/docker-hub-mirror.html) 
> - 想法：解决小白在不熟悉docker、不会代理拉取镜像或者内网情况下上传镜像的问题
> - 思路：简单通过已有的离线镜像包([如何制作/获取离线镜像包](#))，直接通过UI上传

![](./docs/image-upload-portal.jpg)

# 二、项目使用和注意

## 2.1 运行项目

**资源下载**
- 二进制工具下载: [image-upload-portal](https://github.com/404name/image-upload-portal/releases/latest)
- 常用离线镜像包下载: [nginx、mysql、echo-server](https://github.com/404name/image-upload-portal/releases/latest)


**项目运行**

- **linux**: ./image-upload-portal server --port=8088
- **windows**: ./image-upload-portal.exe server --port=8088
- **docker**: mkdir -p /data/uploads && chmod -R 777 /data/uploads && docker run -d --name image-upload-portal -p 8088:8088 -v /data/uploads:/app/uploads 404name/image-upload-portal:latest
- **k8s**: kubectl apply -f ./deploy.yaml

## 2.2 功能

- 支持上传harbor / 阿里云
- 支持UI模式和命令行两种模式

## 2.3 如何制作离线镜像包

> [tip] 也可以参考[github-action](./github/workflows/go-binary-release.yml)里面写入需要拉取的镜像，自动通过release发布离线包(这个后续会单独提供一个分支做这个)

- 国内镜像代理网站（支持大部镜像拉取）：https://docker.aityp.com/
- 国内dockerhub代理（支持全部镜像拉取）：https://m.daocloud.io

1. 拉取镜像：docker pull m.daocloud.io/library/nginx:latest
2. 镜像打tag: docker tag m.daocloud.io/librarynginx:latest nginx:latest
3. 保存离线镜像包：docker save -o nginx.tar nginx:latest
4. 使用这个nginx.tar就可以通过UI直接上传到镜像仓库了

都有docker了，为啥不用docker推送 ==> 有的内网环境没有docker，下载也比较麻烦，而且离线镜像包可以一开始就制作好。

# 三、项目开发和维护

**提交issue**
- 提交bug：https://github.com/404name/image-upload-portal/issues/new/choose
- 提交需求：https://github.com/404name/image-upload-portal/issues/new/choose


**本地开发**

- go mod tidy
- go run ./ server

- docker build -t image-upload-portal:latest  .
- docker run -d --name image-upload-portal -p 8088:8088 image-upload-portal:latest


**实现热加载**

- go get -u github.com/cosmtrek/air
- air init
- air




**TODO**

- [x] dockerfile打包
- [ ] 流水线自动打包linux+windows+docker仓库 + 自动发布tag
- [ ] YAML适配
- [x] 支持阿里云推送
- [ ] 支持华为云和dockerhub等标准场景（网络限制，测试不了dockerhub）
- [ ] 支持默认追加https://
- [ ] 支持上传成功后提示 
- [ ] 支持界面下载镜像和对接harbor
- [ ] 解决华为云推送的时候授权问题华为云的scope="repository:name404/alist:" 似乎需要适配下
```
2025/01/08 13:02:05 push.go:243: [INFO] Received Www-Authenticate header: Bearer realm="https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/",service="dockyard",scope="repository:name404/alist:"
2025/01/08 13:02:05 auth.go:107: [INFO] Parsed Www-Authenticate header - realm: https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/, service: dockyard, scope: repository:name404/alist:,push
2025/01/08 13:02:05 auth.go:22: [INFO] Parsed Www-Authenticate header - realm: https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/, service: dockyard, scope: repository:name404/alist:,push
2025/01/08 13:02:05 auth.go:31: [INFO] Constructed token request data: map[password:[355d42dfa6a82ad2ea20f69d4f846522f59b3a4d116b2180115c5a789dea9cf9] scope:[repository:name404/alist:,push] service:[dockyard] username:[cn-north-4@9OU2ASO8F9VIDXZAXKC6]]
2025/01/08 13:02:05 auth.go:44: [INFO] Sending token request to https://swr.cn-north-4.myhuaweicloud.com/swr/auth/v2/registry/auth/
2025/01/08 13:02:05 auth.go:53: [INFO] Received token response with status code: 404
2025/01/08 13:02:05 auth.go:57: [ERROR] Token request failed with status code: 404

```

# 四、关于

- 代码前端使用的html+axios+tailwind，后端使用go；为了快速实现功能，代码可能没有太规范，有想法一起开发的欢迎提交PR
- GO推送镜像代码参考：https://github.com/silenceper/docker-tar-push （改造实现了指定项目组 + 支持阿里云仓库认证）
- HOW-TO-SOS维护项目: https://howtosos.eryajf.net/
- github推拉代码失败：https://whatismyipaddress.com/hostname-ip


**contributors**

<a href="https://github.com/404name/image-upload-portal/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=404name/image-upload-portal" />
</a>

## Star History Chart


[![Star History Chart](https://api.star-history.com/svg?repos=404name/image-upload-portal&type=Date)](https://star-history.com/#404name/image-upload-portal&Date)

## License

[MIT](./LICENSE).