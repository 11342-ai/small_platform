package Note

import (
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"platfrom/database"
	"platfrom/service/Note"
	"strconv"
)

// RootGetAllNotes 管理员获取所有笔记列表
func RootGetAllNotes(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 可选：按用户ID筛选
	userID, _ := strconv.ParseUint(c.DefaultQuery("user_id", "0"), 10, 32)

	noteService := Note.GlobalNoteService
	notes, total, err := noteService.RootGetAllNotes(uint(userID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取笔记列表失败: " + err.Error()})
		return
	}

	// 关联查询用户名（批量查询优化）
	var userIDs []uint
	for _, note := range notes {
		userIDs = append(userIDs, note.UserID)
	}

	// 查询所有相关用户
	var users []database.User
	database.DB.Where("id IN ?", userIDs).Find(&users)

	// 构建用户ID到用户名的映射
	userMap := make(map[uint]string)
	for _, user := range users {
		userMap[user.ID] = user.Username
	}

	// 转换为响应格式
	var noteResponses []database.AdminNoteResponse
	for _, note := range notes {
		noteResponses = append(noteResponses, database.AdminNoteResponse{
			ID:        note.ID,
			UserID:    note.UserID,
			Username:  userMap[note.UserID],
			Title:     note.Title,
			Content:   note.Content,
			Tags:      note.Tags,
			Category:  note.Category,
			IsPublic:  note.IsPublic,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		})
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, database.AdminNoteListResponse{
		Notes:      noteResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// RootGetNoteByID 管理员查看笔记详情
func RootGetNoteByID(c *gin.Context) {
	noteIDStr := c.Param("id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的笔记ID"})
		return
	}

	noteService := Note.GlobalNoteService
	note, err := noteService.RootGetNoteByID(uint(noteID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	// 查询用户信息
	var user database.User
	if err := database.DB.First(&user, note.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, database.AdminNoteResponse{
		ID:        note.ID,
		UserID:    note.UserID,
		Username:  user.Username,
		Title:     note.Title,
		Content:   note.Content,
		Tags:      note.Tags,
		Category:  note.Category,
		IsPublic:  note.IsPublic,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	})
}

// RootDeleteNote 管理员删除笔记
func RootDeleteNote(c *gin.Context) {
	noteIDStr := c.Param("id")
	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的笔记ID"})
		return
	}

	noteService := Note.GlobalNoteService
	if err := noteService.RootDeleteNote(uint(noteID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除笔记失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "笔记删除成功",
		"note_id": noteID,
	})
}
