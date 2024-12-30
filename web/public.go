package web

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed index.html static/*
var staticFiles embed.FS

func Server() {
	router := gin.Default()
	// 首页接口
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})
	// 静态文件目录
	router.StaticFS("/files", http.FS(staticFiles))
	router.StaticFS("/static", http.FS(staticFiles))

	router.POST("/upload", uploadHandler)
	router.GET("/images", getImagesHandler)
	
	router.Run(":8080")
}

func uploadHandler(c *gin.Context) {
	repo := c.PostForm("repo")
	username := c.PostForm("username")
	password := c.PostForm("password")

	caFile, err := c.FormFile("caFile")
	if err != nil {
		log.Println("获取CA证书失败:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取CA证书失败"})
		return
	}

	imageFile, err := c.FormFile("imageFile")
	if err != nil {
		log.Println("获取镜像包失败:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取镜像包失败"})
		return
	}

	log.Printf("镜像仓库地址: %s\n", repo)
	log.Printf("账号: %s\n", username)
	log.Printf("密码: %s\n", password)
	log.Printf("CA证书: %s\n", caFile.Filename)
	log.Printf("离线镜像包: %s\n", imageFile.Filename)

	// 模拟存储文件
	c.SaveUploadedFile(caFile, "./uploads/"+caFile.Filename)
	c.SaveUploadedFile(imageFile, "./uploads/"+imageFile.Filename)

	c.JSON(http.StatusOK, gin.H{"message": "文件上传成功"})
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