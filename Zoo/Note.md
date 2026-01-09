```
type Note struct {
    gorm.Model
    UserID   uint     `gorm:"index;not null"`
    Title    string   `gorm:"size:255;not null" json:"title"`
    Content  string   `gorm:"type:text;not null" json:"content"`
    Tags     []string `gorm:"type:text" json:"tags"`        //
使用text类型存储JSON
    Category string   `gorm:"size:100;default:'未分类'"
     json:"category"`
    IsPublic bool     `gorm:"default:false" json:"is_public"`
}
```

```
UserID:     用户ID，关联到用户表的ID（外键）
Title:      笔记标题，最大长度255字符，必填
Content:    笔记正文内容，使用text类型存储
Tags:       标签数组，用于分类和组织笔记（JSON格式存储）
Category:   分类名称，默认值为"未分类"，最大长度100字符
IsPublic:   是否公开，默认false（仅创建者可见）
CreatedAt、UpdatedAt、DeletedAt: 由gorm.Model自动管理的时间戳
```

```
 type NoteServiceInterface interface {
     CreateNote(note *database.Note) error
     UpdateNote(UserID uint, id uint, note *database.Note) error
     DeleteNote(UserID uint, id uint) error
     GetNoteByID(UserID uint, id uint) (*database.Note, error)
     GetAllNotes(UserID uint) ([]database.Note, error)
     GetNotesByCategory(UserID uint, category string) ([]database.Note, error)
     GetNotesByTag(UserID uint, tag string) ([]database.Note,error)
     SearchNotes(UserID uint, keyword string) ([]database.Note,error)
 }
```

```
CreateNote        创建新笔记，验证标题非空后保存到数据库
UpdateNote        更新指定笔记，先验证笔记存在且属于该用户，再更新字段
DeleteNote        删除指定笔记，先验证笔记存在且属于该用户
GetNoteByID       根据ID获取单个笔记详情，验证用户权限
GetAllNotes       获取用户的所有笔记列表，按创建时间倒序排列
GetNotesByCategory 根据分类名称筛选用户的笔记
GetNotesByTag     根据标签筛选用户的笔记（使用LIKE查询JSON数组）
SearchNotes       根据关键词搜索笔记（标题和内容全文搜索，不区分大小写）
```

| 路由                            | 方法   | 功能说明         | 是否受到保护 |
|:------------------------------|:-----|:-------------|:-------|
| /api/notes/                   | GET  | 获取当前用户的所...  | 是      |
| /api/notes/:id                | GET  | 根据ID获取单个笔... | 是      |
| /api/notes/                   | POST | 创建新笔记        | 是      |
| /api/notes/:id                | PUT  | 更新指定笔记       | 是      |
| /api/notes/:id                | D... | 删除指定笔记       | 是      |
| /api/notes/category/:category | GET  | 根据分类获取笔记     | 是      |
| /api/notes/tag/:tag           | GET  | 根据标签获取笔记     | 是      |
| /api/notes/search/:keyword    | GET  | 搜索笔记（标题和...  | 是      |

补充说明

1. 用户权限验证：所有路由均通过JWT中间件保护，从请求上下文中提取user_id，确保用户只能操作自己的笔记。

2. **数据验证**：
    - 创建和更新笔记时，title字段为必填项，不可为空
    - category字段在请求中为必填项，提供默认值"未分类"
    - tags字段为可选，支持传入字符串数组

3. **搜索功能**：
    - 支持标题和内容的全文搜索，不区分大小写
    - 使用SQL的LIKE操作符进行模糊匹配
    - 搜索关键词前后自动添加%通配符

4. **标签查询**：
    - 由于SQLite不支持原生数组查询，标签使用JSON数组格式存储
    - 查询时使用LIKE '%"标签"%'模式匹配JSON字符串
    - 支持多个标签的灵活组织

5. **分类管理**：
    - 分类为简单的字符串字段，无需预定义分类列表
    - 用户可自由创建和管理自己的分类体系

6. **公开性控制**：
    - IsPublic字段控制笔记的可见性
    - 当前版本仅支持私有笔记（IsPublic: false）
    - 未来可扩展公开笔记的分享功能

7. **排序规则**：
    - 所有列表查询默认按created_at倒序排列（最新笔记在前）
    - 确保用户始终看到最近创建或更新的笔记

8. **错误处理**：
    - 统一的HTTP状态码返回（401未授权、400参数错误、404资源不存在、500服务器错误）
    - 详细的错误信息返回，便于前端展示和调试

9. **数据库索引**：
    - UserID字段建立索引，提高用户相关查询性能
    - 建议根据实际查询模式，考虑为category、created_at等字段添加索引

10. **扩展性考虑**：
    - 当前设计支持基本的CRUD和分类/标签功能
    - 可轻松扩展支持：笔记版本历史、笔记协作、富文本编辑、附件上传等功能