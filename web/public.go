package web

import (
	"docker-tar-push-ui/pkg/push"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/olahol/melody"
	"github.com/silenceper/log"
)

//go:embed index.html test.html static/*
var staticFiles embed.FS

var (
	uploadDir = "./uploads" // 工作路径
	mu        sync.Mutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

func Server(port string) {
	r := gin.Default()
	log.SetLogLevel(log.Level(log.LevelInfo))
	// 首页接口
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})
	// 静态文件目录
	// r.StaticFS("/files", http.FS(staticFiles))
	r.StaticFS("/static", http.FS(staticFiles))

	r.POST("/upload", uploadHandler)
	r.GET("/files", getImagesHandler)
	r.DELETE("/files", deleteImagesHandler)

	// WebSocket 路由
	m := melody.New() // melody用于实现WebSocket功能
	r.GET("/webterminal", func(c *gin.Context) {
		// 在连接建立后，发送帮助信息
		m.HandleConnect(func(s *melody.Session) {
			log.Infof("New WebSocket connection established: %v", s)
			// 发送帮助信息
			if err := s.Write([]byte(help())); err != nil {
				log.Errorf("Failed to send help message: %v", err)
			}
		})
		m.HandleMessage(func(s *melody.Session, msg []byte) { // 处理来自WebSocket的消息
			log.Infof("Received message: %s", msg)
			if err := handleCommand(s, string(msg)); err != nil {
				log.Errorf("执行命令出错: %v", err)
				s.Write([]byte(fmt.Sprintf("[ERROR]: %s\n", err)))
			}
		})
		m.HandleRequest(c.Writer, c.Request) // 访问 /webterminal 时将转交给melody处理
	})
	r.Run(":" + port)
}

func uploadHandler(c *gin.Context) {
	imageFile, err := c.FormFile("file")
	if err != nil {
		log.Errorf("获取镜像包失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取镜像包失败"})
		return
	}
	log.Infof("离线镜像包: %s\n", imageFile.Filename)
	c.SaveUploadedFile(imageFile, path.Join(uploadDir, imageFile.Filename))
	c.JSON(http.StatusOK, gin.H{"message": "文件上传成功" + imageFile.Filename})
}

func getImagesHandler(c *gin.Context) {
	var records []map[string]string
	files, err := os.ReadDir(uploadDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() {
				record := map[string]string{
					"name":    file.Name(),
					"address": path.Join(uploadDir, file.Name()),
				}
				records = append(records, record)
			}
		}
	}
	c.JSON(http.StatusOK, records)
}

func deleteImagesHandler(c *gin.Context) {
	// 确认要删除的目录
	dirPath := uploadDir

	// 删除目录及其内容
	err := os.RemoveAll(dirPath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete files: %v", err)
		return
	}

	// 返回成功响应
	c.String(http.StatusOK, "All files deleted successfully")
}

func handleCommand(s *melody.Session, command string) error {
	mu.Lock()
	defer mu.Unlock()
	// 按空格切割命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	cmd := parts[0]

	// 如果是退出指令，则优先判断
	if cmd == "exit" {
		s.Set("taskInProgress", false)
		return nil
	}
	// 检查当前任务是否正在运行（只有长连接的请求，才会设置这个变量）
	if inProgress, exists := s.Get("taskInProgress"); exists && inProgress.(bool) {
		return fmt.Errorf("请等待上一个命令执行完毕")
	}

	switch cmd {
	case "docker-tar-push":
		if len(parts) < 7 {
			return s.Write([]byte("请参考：docker-tar-push 镜像包 镜像前缀 镜像地址 账号 密码 ture\n")) // 发送帮助信息
		}
		log.Infof("离线镜像包: %s\n", parts[1])
		log.Infof("镜像仓库地址: %s\n", parts[2])
		log.Infof("镜像前缀: %s\n", parts[3])
		log.Infof("账号: %s\n", parts[4])
		// log.Infof("密码: %s\n", parts[5])
		log.Infof("跳过HTTPS验证: %s\n", parts[6])
		go func() {
			s.Set("taskInProgress", true)
			defer s.Set("taskInProgress", false)
			skipSSLVerify := false
			if parts[6] == "true" {
				skipSSLVerify = true
			}

			imagePush := push.NewImagePush(parts[1], parts[2], parts[3], parts[4], parts[5], skipSSLVerify, s)
			imagePush.Push()
		}()
		return s.Write([]byte("推送中\n")) // 发送帮助信息
	case "ls":
		return s.Write([]byte(listFiles(uploadDir)))
	case "help":
		return s.Write([]byte(help()))
	default:
		return s.Write([]byte("暂不支持的指令：" + help()))
		// TODO后续支持自定义的代码执行
		// return executeDockerTarPush(s, parts) // 将参数传递给执行函数
	}
}

func help() string {
	return `WELCOME 欢迎使用 镜像UI上传工具
 _____ __  __          _____ ______      _    _ _____  _      ____          _____  
|_   _|  \/  |   /\   / ____|  ____|    | |  | |  __ \| |    / __ \   /\   |  __ \ 
  | | | \  / |  /  \ | |  __| |__ ______| |  | | |__) | |   | |  | | /  \  | |  | |
  | | | |\/| | / /\ \| | |_ |  __|______| |  | |  ___/| |   | |  | |/ /\ \ | |  | |
 _| |_| |  | |/ ____ \ |__| | |____     | |__| | |    | |___| |__| / ____ \| |__| |
|_____|_|  |_/_/    \_\_____|______|     \____/|_|    |______\____/_/    \_\_____/ 
Available commands:
- help: Show this help message
- ls: List files in the upload directory
- docker-tar-push <args>: Execute docker-tar-push with the provided arguments
- exit: 退出上一个命令
`
}

func listFiles(dir string) []byte {
	files, err := os.ReadDir(dir)
	if err != nil {
		return []byte(fmt.Sprintf("failed to list files: %v", err))
	}

	var fileList string
	for _, file := range files {
		// 获取文件的绝对路径
		filePath := filepath.Join(uploadDir, file.Name())
		fileList += filePath + "\n"
	}
	return []byte(fileList)
}

func executeDockerTarPush(s *melody.Session, runArgs []string) error {
	//TODO 解决这里卡死的问题
	// 分割命令和参数
	// runArgs := []string{"C:\\Users\\User\\go\\src\\mq.code.sangfor.org\\12626\\docker-tar-push-ui\\docker-tar-push.exe"} // 假设 docker-tar-push.exe 在当前目录下
	// // if runtime.GOOS == "windows" {
	// // 	log.Infof("Windows")
	// // 	cmd = exec.Command("C:\\Windows\\System32\\cmd.exe") // Windows 使用 cmd
	// // } else {
	// // 	log.Infof("Linux or other OS")
	// // 	cmd = exec.Command("bash") // Linux 使用 bash
	// // }
	// runArgs = append(runArgs, "docker-tar-push")
	// runArgs = append(runArgs, args...) // 将用户输入的命令参数添加到 args 中

	// 创建命令
	cmd := exec.Command(runArgs[0], runArgs[1:]...) // 使用当前目录下的 docker-tar-push.exe
	cmd.Dir = uploadDir                             // 设置工作目录为 /upload

	// 创建管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %s", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %s", err)
	}

	// 实时读取输出
	go func() {
		defer stdout.Close()
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				// 将输出发送到 WebSocket
				// 这里需要将输出发送到 WebSocket 连接
				// 假设有一个全局的 WebSocket 连接变量 conn
				s.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer stderr.Close()
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				// 将错误输出发送到 WebSocket
				s.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %s", err)
	}

	return nil
}
