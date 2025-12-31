对于这个项目，我打算借助那个openai接口实现将一下LLM的api进行接入，然后我打算创建如下的**数据库**

********************************

**LLM聊天服务方面的**

**数据库ChatSession**
&emsp;&emsp;用于存储用户**聊天会话的信息**

```
type ChatSession struct {
	SessionID    string    `gorm:"primaryKey;size:50"`
	UserID       uint      `gorm:"index;not null"`
	Title        string    `gorm:"size:200"`
	ModelName    string    `gorm:"not null;default:''"`
	MessageCount int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}
```

```
SessionID: 会话的唯一标识，由系统生成（例如 session_1704067200000000000）
UserID: 用户在那个用户数据库的唯一标识，关联到User表的ID
Title: 会话标题，可由用户的第一条消息自动生成或手动设置
ModelName: 该会话所使用的模型名称（如 deepseek-chat）
MessageCount: 会话中消息的总数（包括用户和AI的）
CreatedAt: 会话创建时间
UpdatedAt: 会话最后更新时间（发送或接收消息时更新）
```

**数据库ChatMessage**
&emsp;&emsp;用于存储**聊天消息的内容**

```
type ChatMessage struct {
	gorm.Model
	SessionID string `gorm:"index;not null;size:50"`
	Role      string `gorm:"size:20;not null"` // user, assistant, system
	Content   string `gorm:"type:text"`
}
```

```
SessionID: 关联的会话ID，指向ChatSession表的SessionID
Role: 消息角色，可选值：user（用户）、assistant（AI助手）、system（系统提示）
Content: 消息的文本内容
CreatedAt、UpdatedAt、DeletedAt: 由gorm.Model自动管理的时间戳
```

**数据库UploadedFile**
&emsp;&emsp;用于存储用户**上传的文件信息**，支持在聊天中附加文件内容

```
type UploadedFile struct {
	gorm.Model
	SessionID   string `gorm:"index;not null"` // 关联的会话ID
	FileName    string `gorm:"not null"`       // 原文件名
	FilePath    string `gorm:"not null"`       // 存储路径
	FileSize    int64  `gorm:"not null"`       // 文件大小
	FileType    string `gorm:"not null"`       // 文件类型
	Content     string `gorm:"type:text"`      // 文件内容（文本文件）
	IsProcessed bool   `gorm:"default:false"`  // 是否已处理
}
```

```
SessionID: 关联的会话ID
FileName: 用户上传的原始文件名
FilePath: 文件在服务器上的存储路径（如 ./uploads/session_xxx_abc.txt）
FileSize: 文件大小（字节）
FileType: 文件扩展名（如 .txt、.py）
Content: 如果是文本文件，保存其内容；否则为空
IsProcessed: 标记该文件是否已被处理（例如是否已读入消息）
```

**Persona配置（YAML文件，非数据库表）**
&emsp;&emsp;用于定义不同的人格（系统提示词），保存在 `style.yaml` 中

```
type Persona struct {
	Name    string `yaml:"name" json:"name"`
	Content string `yaml:"content" json:"content"`
}
```

```
Name: 人格名称（如 "默认助手"、"编程专家"）
Content: 对应的系统提示词内容（如 "你是一个有帮助的AI助手..."）
```

********************************

**LLM聊天服务方面的接口**

```
type ChatServiceInterface interface {
	CreateChatSession(sessionID, modelName string, UserId uint) (*database.ChatSession, error)
	SaveChatMessage(sessionID, role, content string, UserId uint) error
	GetChatMessages(sessionID string) ([]openai.ChatCompletionMessage, error)
	GetChatSessions(UserId uint) ([]database.ChatSession, error)
	GetChatSession(sessionID string, UserId uint) (*database.ChatSession, error)
	DeleteChatSession(sessionID string) error
	UpdateSessionTitle(sessionID, title string) error
}
```

```
CreateChatSession ====》创建聊天会话，如果已存在则返回现有会话
SaveChatMessage ====》保存一条聊天消息到数据库，并更新会话的消息计数
GetChatMessages ====》获取指定会话的所有消息，按时间顺序返回
GetChatSessions ====》获取指定用户的所有聊天会话列表，按更新时间倒序排列
GetChatSession ====》获取指定会话的详细信息（需验证用户权限）
DeleteChatSession ====》删除聊天会话及其所有关联的消息（事务操作）
UpdateSessionTitle ====》更新会话的标题
```

```
type LLMSessionInterface interface {
	SetSessionID(sessionID string)
	GetMessages() []openai.ChatCompletionMessage
	SendMessage(message string) (string, error)
	SendMessageStream(message string, onChunk func(chunk string) error) (string, error)
	SetSystemPrompt(prompt string)
}
```

```
SetSessionID ====》设置会话ID（用于关联数据库记录）
GetMessages ====》获取当前内存中的消息历史
SendMessage ====》同步发送消息到AI模型并返回完整响应
SendMessageStream ====》流式发送消息，支持实时返回响应片段
SetSystemPrompt ====》设置系统提示词（用于切换人格）
```

```
type UserAPIServiceInterface interface {
	// API配置管理
	CreateAPI(userID uint, api *database.UserAPI) (*database.UserAPI, error)
	GetAPIByID(apiID uint) (*database.UserAPI, error)
	GetAPIByName(userID uint, apiName string) (*database.UserAPI, error)
	GetAPIByModelName(userID uint, modelName string) (*database.UserAPI, error)
	GetUserAPIs(userID uint) ([]database.UserAPI, error)
	UpdateAPI(apiID uint, updates map[string]interface{}) error
	DeleteAPI(apiID uint) error

	// API验证与选择
	TestAPIConnection() (bool, error)
	GetFirstAvailableAPI(userID uint) (*database.UserAPI, error)
}
```

```
（这部分已在用户API配置中描述，此处仅列出与LLM聊天相关的函数）
GetAPIByModelName ====》根据模型名称获取用户的API配置（聊天时选择模型用）
GetFirstAvailableAPI ====》获取用户第一个可用的API配置（用于下拉列表默认值）
```

```
type FileServiceInterface interface {
	SaveFile(file *database.UploadedFile) error
	GetFilesBySession(sessionID string) ([]database.UploadedFile, error)
	GetFileByID(id uint) (*database.UploadedFile, error)
	DeleteFile(id uint) error
	ProcessFileContent(file *database.UploadedFile) (string, error)
}
```

```
SaveFile ====》保存上传的文件信息到数据库
GetFilesBySession ====》获取指定会话的所有上传文件
GetFileByID ====》根据文件ID获取文件详细信息
DeleteFile ====》删除文件（同时删除物理文件）
ProcessFileContent ====》处理文件内容，如果是文本文件则读取内容返回
```

```
type PersonaManagerInterface interface {
	GetPersonaContent(personaName string) string
	GetAvailablePersonas() []string
	SetDefaultPersona(personaName string)
	GetDefaultPersona() string
}
```

```
GetPersonaContent ====》根据人格名称获取对应的系统提示词内容
GetAvailablePersonas ====》获取所有可用的人格名称列表
SetDefaultPersona ====》设置默认人格（当前未在路由中使用）
GetDefaultPersona ====》获取当前默认人格名称
```

```
type CacheServiceInterface interface {
	CacheChatSession(sessionID string, session *database.ChatSession, expiration time.Duration) error
	GetCachedChatSession(sessionID string) (*database.ChatSession, error)
	CacheModelConfig(modelName string, model *database.UserAPI) error
	CacheFullSession(sessionID string, cachedSession *CachedSession, expiration time.Duration) error
	GetCachedFullSession(sessionID string) (*CachedSession, error)
}
```

```
CacheChatSession ====》缓存聊天会话信息到Redis
GetCachedChatSession ====》从Redis获取缓存的会话信息
CacheModelConfig ====》缓存模型配置信息
CacheFullSession ====》缓存完整会话状态（包括消息历史）
GetCachedFullSession ====》获取完整会话状态（用于快速恢复会话）
```

**SessionManager**（非接口，但为核心管理类）
&emsp;&emsp;管理多个聊天会话的生命周期，协调各服务之间的调用

```
主要方法：
GetOrCreateSession ====》获取或创建会话（核心方法，整合了人格、模型配置、缓存、数据库）
GetSession ====》从内存中获取会话（不创建）
SaveMessage ====》保存消息到数据库（委托给ChatService）
DeleteSession ====》从内存和数据库删除会话
GetAvailablePersonas ====》获取可用人格列表（委托给PersonaManager）
```

********************************

**LLM聊天相关路由**

| 路由                            | 负责的功能                                   | 是否受保护 |
|:------------------------------|:----------------------------------------|:------|
| /api/chat/message             | 同步发送聊天消息                               | 是     |
| /api/chat/message/stream      | 流式发送聊天消息（Server-Sent Events）           | 是     |
| /api/chat/session             | 创建新聊天会话（自动生成会话ID）                     | 是     |
| /api/chat/sessions            | 获取当前用户的所有聊天会话列表                       | 是     |
| /api/chat/sessions/:session_id/messages | 获取指定会话的所有历史消息                  | 是     |
| /api/chat/sessions/:session_id | 删除指定会话（及其所有消息和文件）                    | 是     |

**文件管理路由**

| 路由                               | 负责的功能                           | 是否受保护 |
|:---------------------------------|:--------------------------------|:------|
| /api/files/upload                | 上传文件（关联到会话）                   | 是     |
| /api/files/session/:session_id   | 获取指定会话的所有上传文件列表              | 是     |
| /api/files/:file_id              | 删除指定文件（同时删除物理文件）             | 是     |

**人格管理路由**

| 路由                     | 负责的功能               | 是否受保护 |
|:-----------------------|:--------------------|:------|
| /api/personas/         | 获取所有可用的人格名称列表     | 是     |

**用户API配置路由**（已在用户服务中描述，但用于聊天模型选择）

| 路由                          | 负责的功能                                       | 是否受保护 |
|:----------------------------|:--------------------------------------------|:------|
| /api/user/apis              | 获取用户的所有API配置列表                           | 是     |
| /api/user/apis/first        | 获取用户第一个可用的API配置（用于下拉列表默认值）               | 是     |
| /api/user/apis/:name        | 根据API名称获取具体的API配置（可用于前端选择模型）             | 是     |
| /api/user/apis/:id          | 更新或删除API配置（PUT/DELETE）                     | 是     |

********************************

**补充说明**

1.  **会话生命周期**：会话由`SessionManager`统一管理，支持从缓存（Redis）快速恢复，降级时直接从数据库加载。
2.  **人格切换**：通过`PersonaManager`加载`style.yaml`中定义的人格，系统提示词会在会话创建或切换时设置。
3.  **文件处理**：支持上传文本文件（`.txt`、`.py`、`.go`等），文件内容会被读取并附加到用户消息中。
4.  **流式响应**：使用Server-Sent Events（SSE）实现流式输出，前端可以实时显示AI回复。
5.  **模型选择**：聊天时需指定`model_name`，后端通过`UserAPIService`查找用户对应的API配置（`api_key`和`base_url`）。
6.  **默认行为**：若未指定人格，使用`style.yaml`中的第一个人格；若未指定`base_url`，使用API配置中存储的`BaseURL`。
7.  **缓存策略**：会话信息、模型配置可缓存到Redis，提高响应速度；Redis不可用时自动降级到数据库。