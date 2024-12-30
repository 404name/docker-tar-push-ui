package web

import (
	"embed"
	"fmt"
	"image-upload-portal/pkg/push"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/silenceper/log"
)

//go:embed index.html static/*
var staticFiles embed.FS

var (
    uploadDir = "./uploads" // 工作路径
    mu        sync.Mutex    // 用于保护对 uploadDir 的并发访问
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 允许所有来源
    },
}


func Server() {
	r := gin.Default()
	// 首页接口
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})
	// 静态文件目录
	r.StaticFS("/files", http.FS(staticFiles))
	r.StaticFS("/static", http.FS(staticFiles))

	r.POST("/upload", uploadHandler)
	r.GET("/images", getImagesHandler)

    // WebSocket 路由
    r.GET("/ws", handleWebSocket)
	
	r.Run(":8080")
}

func uploadHandler(c *gin.Context) {
	repo := c.PostForm("repo")
	username := c.PostForm("username")
	password := c.PostForm("password")

	// TODO 实现CA证书逻辑
	// caFile, err := c.FormFile("caFile")
	// if err != nil {
	// 	log.Println("获取CA证书失败:", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "获取CA证书失败"})
	// 	return
	// }

	imageFile, err := c.FormFile("imageFile")
	if err != nil {
		log.Errorf("获取镜像包失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取镜像包失败"})
		return
	}

	log.Infof("镜像仓库地址: %s\n", repo)
	log.Infof("账号: %s\n", username)
	log.Infof("密码: %s\n", password)
	// log.Printf("CA证书: %s\n", caFile.Filename)
	log.Infof("离线镜像包: %s\n", imageFile.Filename)

	// 模拟存储文件
	// c.SaveUploadedFile(caFile, "./uploads/"+caFile.Filename)
	c.SaveUploadedFile(imageFile, "./uploads/"+imageFile.Filename)
	imagePush := push.NewImagePush("./uploads/"+imageFile.Filename, repo, username, password, "library/", true)
	imagePush.Push()
	imageURL := fmt.Sprintf("%s/%s", repo, "library/")
	c.JSON(http.StatusOK, gin.H{"message": "镜像上传成功" + imageURL})
}
func getImagesHandler(c *gin.Context) {
    files, err := os.ReadDir("uploads")
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to read files: %v", err)
        return
    }

    var records []map[string]string
    for _, file := range files {
        if !file.IsDir() {
            record := map[string]string{
                "name":    file.Name(),
                "address": fmt.Sprintf("/files/%s", file.Name()),
            }
            records = append(records, record)
        }
    }

    c.JSON(http.StatusOK, records)
}

func handleWebSocket(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Errorf("Failed to upgrade connection: %v", err)
        return
    }
    defer conn.Close()

    // 发送帮助信息
    conn.WriteMessage(websocket.TextMessage, []byte("Welcome to the Docker Tar Push Terminal!" + help()))

    for {
        // 读取客户端发送的消息
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Errorf("Error reading message: %v", err)
            break
        }

        // 处理命令
        if err := handleCommand(conn, string(msg)); err != nil {
            log.Errorf("Error executing command: ", err)
            conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %s", err)))
            continue
        }
    }
}

func handleCommand(conn *websocket.Conn, command string) error {
    mu.Lock() // 加锁，防止并发访问
    defer mu.Unlock()

    // 按空格切割命令
    parts := strings.Fields(command) // 将命令按空格分割成多个部分
    if len(parts) == 0 {
        return nil // 如果没有命令，直接返回
    }

    cmd := parts[0] // 第一个部分是命令
    args := parts[1:] // 后续部分是参数

    switch cmd {
    case "docker-tar-push":
        return executeDockerTarPush(conn, args) // 将参数传递给执行函数
    case "ls":
        return conn.WriteMessage(websocket.TextMessage, []byte(listFiles(uploadDir))) // 发送文件列表
    default:
        return conn.WriteMessage(websocket.TextMessage, []byte(help())) // 发送帮助信息
    }
}

func help() string {
	return `Available commands:
- help: Show this help message
- ls: List files in the upload directory
- docker-tar-push <args>: Execute docker-tar-push with the provided arguments`
}

func listFiles(dir string) []byte {
    // 获取目录的绝对路径
    absDir, err := filepath.Abs(dir)
    if err != nil {
        return []byte(fmt.Sprintf("failed to get absolute path: %v", err))
    }

    files, err := os.ReadDir(dir)
    if err != nil {
        return []byte(fmt.Sprintf("failed to list files: %v", err))
    }

    var fileList string
    for _, file := range files {
        // 获取文件的绝对路径
        filePath := filepath.Join(absDir, file.Name())
        fileList += filePath + "\n"
    }
    return []byte(fileList)
}

func executeDockerTarPush(conn *websocket.Conn, args []string) error {
    // 分割命令和参数
    runArgs := []string{"C:\\Users\\User\\go\\src\\mq.code.sangfor.org\\12626\\image-upload-portal\\docker-tar-push.exe"} // 假设 docker-tar-push.exe 在当前目录下
	runArgs = append(runArgs, "docker-tar-push")
    runArgs = append(runArgs, args...) // 将用户输入的命令参数添加到 args 中

    // 创建命令
    cmd := exec.Command(runArgs[0], runArgs[1:]...) // 使用当前目录下的 docker-tar-push.exe
    cmd.Dir = uploadDir // 设置工作目录为 /upload

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
                conn.WriteMessage(websocket.TextMessage, buf[:n])
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
                conn.WriteMessage(websocket.TextMessage, buf[:n])
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