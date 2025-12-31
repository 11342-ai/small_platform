package Note

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platfrom/database"
	"platfrom/service/Note"
	"strconv"
)

// SetupNoteRoutes 设置笔记路由
func setupNoteApiRoutes(router *gin.Engine) {
	notes := router.Group("/api/notes")
	{
		notes.GET("/", GetNotes)
		notes.GET("/:id", GetNoteByID)
		notes.POST("/", CreateNote)
		notes.PUT("/:id", UpdateNote)
		notes.DELETE("/:id", DeleteNote)
		notes.GET("/category/:category", GetNotesByCategory)
		notes.GET("/tag/:tag", GetNotesByTag)
		notes.GET("/search/:keyword", SearchNotes)
	}
}

// GetNotes 获取所有笔记
func GetNotes(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}
	notes, err := Note.GlobalNoteService.GetAllNotes(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": notes,
	})

}

func GetNoteByID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID",
		})
		return
	}

	note, err := Note.GlobalNoteService.GetNoteByID(userID.(uint), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": note,
	})

}

// CreateNote 创建笔记
func CreateNote(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	// 定义请求结构体（用于绑定和验证）
	type CreateNoteRequest struct {
		Title    string   `json:"title" binding:"required"`
		Content  string   `json:"content" binding:"required"`
		Tags     []string `json:"tags"`
		Category string   `json:"category" binding:"required"`
		IsPublic bool     `json:"is_public"`
	}

	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
		})
		return
	}

	note := database.Note{
		UserID:   userID.(uint),
		Title:    req.Title,
		Content:  req.Content,
		Tags:     req.Tags,
		Category: req.Category,
		IsPublic: req.IsPublic,
	}

	if err := Note.GlobalNoteService.CreateNote(&note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"data":    note,
	})

}

// UpdateNote 更新笔记
func UpdateNote(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID",
		})
		return
	}

	// 定义更新请求DTO（与创建类似，但可能有些字段可选）
	type UpdateNoteRequest struct {
		Title    string   `json:"title" binding:"required"`
		Content  string   `json:"content" binding:"required"`
		Tags     []string `json:"tags"`
		Category string   `json:"category" binding:"required"`
		IsPublic bool     `json:"is_public"`
	}

	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
		})
		return
	}

	// 构建要更新的笔记数据
	updatedNote := database.Note{
		Title:    req.Title,
		Content:  req.Content,
		Tags:     req.Tags,
		Category: req.Category,
		IsPublic: req.IsPublic,
	}

	if err := Note.GlobalNoteService.UpdateNote(userID.(uint), uint(id), &updatedNote); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})

}

// DeleteNote 删除笔记
func DeleteNote(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID",
		})
		return
	}

	if err := Note.GlobalNoteService.DeleteNote(userID.(uint), uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})

	return
}

// GetNotesByCategory 根据分类获取笔记
func GetNotesByCategory(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	category := c.Param("category")
	notes, err := Note.GlobalNoteService.GetNotesByCategory(userID.(uint), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": notes,
	})

}

// GetNotesByTag 根据标签获取笔记
func GetNotesByTag(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	tag := c.Param("tag")
	notes, err := Note.GlobalNoteService.GetNotesByTag(userID.(uint), tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": notes,
	})
}

// SearchNotes 搜索笔记
func SearchNotes(c *gin.Context) {

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	keyword := c.Param("keyword")
	notes, err := Note.GlobalNoteService.SearchNotes(userID.(uint), keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": notes,
	})
}
