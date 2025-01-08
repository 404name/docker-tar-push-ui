# 镜像UI上传工具 

image-upload-portal 解决部分场景无法使用命令行上传镜像的问题

![](./docs/image-upload-portal.jpg)

# install

go mod tidy
go run ./ server

docker build -t image-upload-portal:latest  .
docker run -d --name image-upload-portal -p 8088:8088 image-upload-portal:latest

**实现热加载**
go get -u github.com/cosmtrek/air
air init
air

# docker-tar-push
push your docker tar archive image without docker

## 功能

- 支持上传harbor / 阿里云
- 支持UI模式和命令行两种模式


**用法一**  
```shell
docker-tar-push alpine:latest --registry=http://localhost:5000
```

**用法二**  
例如将 `docker save python:3.0 > python-3.10.tar` 镜像文件推送harbor仓库, 这时需要存放至 harbor仓库 library 项目中，使用下面参数 `--image-prefix=library/` 即可。   
```shell
docker-tar-push /镜像目录路径 --registry=http://harbor.harbor.svc --username=admin --password=Harbor12345 --image-prefix=library/
go run ./ docker-tar-push ./uploads/whoami.tar.gz --registry=https://10.113.66.245 --username=admin --password=Harbor-12345 --skip-ssl-verify=true --image-prefix=library/

docker-tar-push \uploads\image-upload-portal.rar https://10.113.66.245 admin Harbor-12345 library/
```
当我们从仓库下载镜像时，它的完整名称为: `docker pull harbor.harbor.svc/library/python:3.0`  

## 编译

```sh
go build -o bin/docker-tar-push cmd/docker-tar-push/main.go
```


# TODO

- [x] dockerfile打包
- [ ] YAML适配
- [x] 支持阿里云推送

# 说明

- 底层GO推送镜像代码来自：https://github.com/silenceper/docker-tar-push
- 基于这个实现了指定项目组 + 支持阿里云仓库认证