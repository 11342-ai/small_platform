package LLM_Chat

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"platfrom/database"
	LLM_Chat_Service "platfrom/service/LLM_Chat"
	"strings"
)

func setupFileRoutes(router *gin.Engine) {
	files := router.Group("/api/files")
	{
		files.POST("/upload", UploadFile())
		files.GET("/session/:session_id", GetSessionFiles())
		files.DELETE("/:file_id", DeleteFile())
	}
}

func UploadFile() gin.HandlerFunc {
	fileService := LLM_Chat_Service.GlobalFileService
	return func(c *gin.Context) {
		sessionID := c.PostForm("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id 不能为空"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败: " + err.Error()})
			return
		}

		// 创建上传目录
		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败: " + err.Error()})
			return
		}

		// 生成唯一文件名
		fileExt := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%s_%s%s", sessionID, generateRandomString(8), fileExt)
		filePath := filepath.Join(uploadDir, fileName)

		// 保存文件
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败: " + err.Error()})
			return
		}

		// 读取文件内容（如果是文本文件）
		var fileContent string
		textExtensions := []string{".txt", ".py", ".go", ".c", ".cpp", ".h", ".hpp", ".js", ".ts", ".java", ".html", ".css", ".md", ".json", ".xml", ".yaml", ".yml"}

		ext := strings.ToLower(fileExt)
		isTextFile := false
		for _, textExt := range textExtensions {
			if ext == textExt {
				isTextFile = true
				break
			}
		}

		if isTextFile {
			content, err := os.ReadFile(filePath)
			if err == nil {
				fileContent = string(content)
			}
		}

		// 保存到数据库
		uploadedFile := &database.UploadedFile{
			SessionID:   sessionID,
			FileName:    file.Filename,
			FilePath:    filePath,
			FileSize:    file.Size,
			FileType:    filepath.Ext(file.Filename),
			Content:     fileContent,
			IsProcessed: false,
		}

		if err := fileService.SaveFile(uploadedFile); err != nil {
			// 删除已上传的文件
			os.Remove(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件信息失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "文件上传成功",
			"file_id":   uploadedFile.ID,
			"file_name": uploadedFile.FileName,
			"file_type": uploadedFile.FileType,
		})
	}
}

func generateRandomString(length int) string {
	// 简单的随机字符串生成，实际使用时可以改进
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = byte(65 + (i % 26)) // A-Z
	}
	return string(bytes)
}

func GetSessionFiles() gin.HandlerFunc {
	fileService := LLM_Chat_Service.GlobalFileService
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		files, err := fileService.GetFilesBySession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件列表失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": files})
	}
}

func DeleteFile() gin.HandlerFunc {
	fileService := LLM_Chat_Service.GlobalFileService
	return func(c *gin.Context) {
		fileID := c.Param("file_id")

		// 这里需要将字符串fileID转换为uint
		var id uint
		_, err := fmt.Sscanf(fileID, "%d", &id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		// 先获取文件信息以便删除物理文件
		file, err := fileService.GetFileByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}

		// 删除物理文件
		if file.FilePath != "" {
			os.Remove(file.FilePath)
		}

		// 删除数据库记录
		if err := fileService.DeleteFile(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "文件删除成功"})
	}
}
