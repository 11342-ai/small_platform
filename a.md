对于这个项目，我打算借助那个openai接口实现将一下LLM的api进行接入，然后我打算创建如下的**数据库**

********************************

**用户服务方面的**

**数据库User**   
&emsp;&emsp;用于存储用户**个人的信息**

```
// User 用户数据存储结构
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null;size:50"`
	PasswordHash string `gorm:"not null;size:255"`
	Email        string `gorm:"size:100"`
	LastLogin    time.Time
}
```

```
Username:用户名
PasswordHash：经过哈希加密后的密码
Email：用户email
LastLogin：最后登录的时间
```

**用户服务方面的接口**

```
// UserService 用户服务接口
type UserService interface {
	CreateUser(req database.RegisterRequest) (*database.User, error)
	GetUserByUsername(username string) (*database.User, error)
	GetUserByID(id uint) (*database.User, error)

	// SendVerificationCode 验证码相关功能
	SendVerificationCode(username, codeType string) (*database.VerificationCode, error)
	VerifyCode(username, code, codeType string) (bool, error)

	// ResetPassword 密码相关功能
	ResetPassword(username, code, newPassword string) error            // 忘记密码重置（通过验证码）
	UpdatePassword(userID uint, oldPassword, newPassword string) error // 修改密码（需要旧密码）

	// StartCleanupTask 启动验证码清理任务
	StartCleanupTask()
}
```

```
CreateUser ====》创建用户-->  对于那个表而言，只是一个插入操作而已  
GetUserByUsername ====》通过用户名来寻找用户信息  
GetUserByID====》通过用户ID来寻找那个用户信息
SendVerificationCode ====》模拟发送验证码的过程，目前只是打印在控制台而已
VerifyCode ====》验证验证码
ResetPassword  ====》忘记密码功能
UpdatePassword====》修改密码功能
StartCleanupTask() ====》一个清理验证码的定时任务
```

| 路由                   | 负责的功能           | 是否受保护 |
|:---------------------|:----------------|:------|
| /register            | 注册的路由           | 否     |
| /login               | 登录的路由           | 否     |
| /logout              | 推出登录的路由         | 否     |
| /auth/send-code      | 验证时发送验证码请求的路由   | 否     |
| /auth/verify-code    | 验证时填入验证码后路由     | 否     |
| /auth/reset-password | 忘记密码的哪个路由       | 否     |
| /profile             | 查看个人资料的路由       | 是     |
| /update-password     | 更新个人密码的路由       | 是     |
| /me                  | 为前端提供更友好的用户信息端点 | 是     |

********************************

**大模型聊天方面的数据库等配置**

**这个是那个promote自我设置的接口**

```
// ConfigManager 配置管理器接口
type ConfigManager interface {
	LoadConfig(filePath string) error
	GetPersona(name string) (*database.Persona, error)
	GetAllPersonas() []database.Persona
	ReloadConfig() error
	GetDefaultPersona() (*database.Persona, error)
}
```

```
LoadConfig( ====》加载并解析配置文件
GetPersona ====》通过那个人格名称获取那个人格具体特点
GetAllPersonas ====》获取所有的用户配置
ReloadConfig ====》重新加载那个人格配置
GetDefaultPersona ====》获取那个默认的人格配置
```

**这个是那个promote自我设置的接口**

```
// FileUploadConfigManager 文件上传配置管理器接口
type FileUploadConfigManager interface {
	LoadConfig(filePath string) error
	GetUploadDir() string
	GetMaxFileSize() int64
	GetAllowedExtensions() []string
	ReloadConfig() error
}
```

```
LoadConfig(filePath string) error ====>加载并解析配置文件
GetUploadDir() string ====>获取那个用户上传地 路径
GetMaxFileSize() ====》 获取我设置地允许文件最大地大小
GetAllowedExtensions() []string ====》获取允许通过地文件类型
ReloadConfig() error ====》重新加载配置
```

**数据库VerificationCode**
&emsp;&emsp;用于存储用户**个人的信息**

```
type VerificationCode struct {
	gorm.Model
	Username  string    `gorm:"not null;size:50;index"`
	Code      string    `gorm:"not null;size:6"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	CodeType  string    `gorm:"size:20"` // 验证码类型: password_reset, register, etc.
}
```

```
Username：用户的用户名
Code:验证码
Used：是否被使用
CodeType：验证码用途
ExpiresAt：有效时间
```

**数据库UserAPI**  
&emsp;&emsp;用于存储用户个人的api，可以对**用户的api**进行管理 //增删改查  

```
// UserAPI 用户API配置
type UserAPI struct {   
    gorm.Model
    UserID    uint   `gorm:"index;not null"`    
    APIName   string `gorm:"size:100;not null"`     
    APIKey    string `gorm:"size:500;not null"` // 加密存储     
    ModelName string `gorm:"size:100"`     
    BaseURL   string `gorm:"size:500"` 
}
```

```
UserID:用户再那个用户数据库的唯一标识  
APIName：用户为某一个api_key取得别名，毕竟用户不可能看着一串数字就行选择的  
APIKey:就是api  
ModelName：模型名称，例如deepsssk-chat等  
BaseURL:例如https://api.deepseek.com/v1等
```

&emsp;&emsp;一个用户可以有多个api_key的,但是某一个api只属于一个用户，这点要注意  
&emsp;&emsp;我个人认为数据库A不需要那个provider,毕竟**github.com/sashabaranov/go-openai**只需要**api_key**, **model**,**base_url**这三个就可以正常连接对应的网站url，使用人家包装好的api了  
&emsp;&emsp;我个人认为，对于那个persona默认的问题，在前端创建新对话时，肯定是有一个选择人格的操作，这时候再载入就行；而且前端选择前也可以将第一个作为默认的嘛！  
&emsp;&emsp;对于那个不是新对话的问题，我们可以读取数据库B当中的persona！<br>
&emsp;&emsp;对于那个默认的api的问题，到时候前端操作时，如果没有特定选择，就直接选择表当中读取的第一个嘛！不用在数据库当中特意指定 // 别加到数据库当中，我不想高太多  
&emsp;&emsp;对于api能不能用，这个交由那个用户就行自我管理就好，其他的我不想管那个api能不能用；且对于什么provider，反正有那个base_url，一下子就看出来了  
&emsp;&emsp;<br>

**用户个人API配置方面地接口**  
&emsp;&emsp;这个主要是针对于那个用户个人api地增删改查而已

```
// UserAPIService 用户API配置服务接口
type UserAPIService interface {
	// API配置管理
	CreateAPI(userID uint, api *database.UserAPI) (*database.UserAPI, error)
	GetAPIByID(apiID uint) (*database.UserAPI, error)
	GetAPIByName(userID uint, apiName string) (*database.UserAPI, error)
	GetUserAPIs(userID uint) ([]database.UserAPI, error)
	UpdateAPI(apiID uint, updates map[string]interface{}) error
	DeleteAPI(apiID uint) error
	GetAPIBySession(sessionID string) (*database.UserAPI, error) ----》我个人认为这个不好，应该可以去除掉，毕竟这个小项目应该保持简洁 // 而且，那个获取session的api是不必要的，它应该是前端选择了配置后传给那个后端的对话，serssion库本身都没有这玩意

	// API验证与选择
	TestAPIConnection() (bool, error)
	GetFirstAvailableAPI(userID uint) (*database.UserAPI, error)

	// 统计
	CountUserAPIs(userID uint) (int64, error)
}
```

```
CreateAPI====》根据那个用户ID等创建一个api而已
GetAPIByID====>创建/插入用户自己地api
GetAPIByName====》通过用户ID和那个api的名字来获取api的详细信息
GetUserAPIs====》通过用户名来获取那个用户的全部api
UpdateAPI====》用于更新那个用户api的名称
DeleteAPI====》用于删除某一个api
GetAPIBySession ====》根据那个对话id来获取那个api的配置
TestAPIConnection ====》测试api的连接
GetFirstAvailableAPI ====》获取用户第一个可用的API配置，就是一堆api当中第一个
CountUserAPIs====》统计用户api的数量----》我个人认为这个不好，应该可以去除掉，毕竟这个小项目应该保持简洁
```

```
```

**数据库ChatSession**
&emsp;&emsp;用于存储用户的对话id，例如，一个对话会有多个消息，但是只有一个id // 有插入、删除、读取、改(改那个**人格**，毕竟如果用户选择了新人格，肯定要有及时反馈的)  

```
// ChatSession 聊天会话
type ChatSession struct {
	SessionID     string `gorm:"primaryKey;size:50"`
	UserID        uint   `gorm:"index;not null"`
	Title         string `gorm:"size:200"`
	PersonaName   string `gorm:"size:100"`
	LastMessageAt time.Time
	MessageCount  int `gorm:"default:0"`
}
```

```
SessionID:会话的id 
UserID:所属用户的ID 
Title:会话名字  
PersonaName:人格  
LastMessageAt:最新说话时间  
MessageCount:用户对话回数  
LastMessageAt:最后对话时间
```

**ChatSessionService 聊天会话服务接口**

```
// ChatSessionService 聊天会话服务接口
type ChatSessionService interface {
	// CreateSession 会话管理
	CreateSession(userID uint, title string, personaName string) (*database.ChatSession, error)
	GetSessionByID(sessionID string) (*database.ChatSession, error)
	GetUserSessions(userID uint) ([]database.ChatSession, error)
	GetUserSessionsByPage(userID uint, page, pageSize int) ([]database.ChatSession, int64, error)
	UpdateSession(sessionID string, updates map[string]interface{}) error
	UpdateSessionTitle(sessionID, title string) error
	UpdateSessionPersona(sessionID, personaName string) error
	DeleteSession(sessionID string) error
	DeleteUserSessions(userID uint) error

	UpdateLastMessageTime(sessionID string) error
	IncrementMessageCount(sessionID string) error
}
```

```
CreateSession ====》创建对话逻辑
GetSessionByID ====》通过那个sessionid来获取那个对话详情
GetUserSessions ====》通过用户id来获取那用户所有的对话
GetUserSessionsByPage ====》当用户对话过多，肯定不是一次性把那个用户对话列表传输过去，分成几部分，分段获取
UpdateSessionTitle ====》更新会话标题
UpdateSessionPersona ====》更新会话人格
DeleteSession ====》删除某一个会话
DeleteUserSessions ====》删除某一个用户的全部会话，例如用户注销时
GetSessionCount ====》获取用户对话数量 ----》我个人认为这个不好，应该可以去除掉，毕竟这个小项目应该保持简洁
UpdateLastMessageTime ====》更新对话的最后更新时间
IncrementMessageCount ====》增加那个对话里面的消息计数----》我个人认为这个不好，应该可以去除掉，毕竟这个小项目应该保持简洁
```


**数据库ChatMessage**   
&emsp;&emsp;用于存储所有的那些对话消息 //只有插入、读取、删除

```
type ChatMessage struct {
	gorm.Model
	SessionID    string `gorm:"index;not null;size:50"`
	UserID       uint   `gorm:"index;not null"`
	Role         string `gorm:"size:20;not null"` // user, assistant, system
	Content      string `gorm:"type:text"`
	MessageOrder int    `gorm:"not null"`
}
```

```
SessionID:
UserID:message所属的用户
Role:用户，如user\system等  
Content：就是那个用户回复的内容 
created_at:创建/插入时间
MessageOrder:那个消息的顺序
```

**ChatMessageService接口方面的东西**

```
// ChatMessageService 聊天消息服务接口
type ChatMessageService interface {
	// CreateMessage 消息管理
	CreateMessage(sessionID string, userID uint, role, content string) (*database.ChatMessage, error)
	// GetMessageByID 消息查询
	GetSessionMessages(sessionID string) ([]database.ChatMessage, error)
	GetSessionMessagesByPage(sessionID string, page, pageSize int) ([]database.ChatMessage, int64, error)
	// DeleteMessage 消息删除
	DeleteSessionMessages(sessionID string) error
	DeleteUserMessages(userID uint) error
	// GetSessionContext 上下文管理
	GetSessionContext(sessionID string, maxMessages int) ([]database.ChatMessage, error)
	// GetNextMessageOrder 消息顺序管理
	GetNextMessageOrder(sessionID string) (int, error)
}
```

```
CreateMessage ====》创建消息
GetSessionMessages ====》获取会话的所有消息
GetSessionMessagesByPage ====》分页获取会话消息毕竟用户连接上后，不可能一次性把所有的消息都传过去
DeleteSessionMessages ====>删除会话的所有消息
GetSessionContext ====》获取会话上下文
GetNextMessageOrder ====》获取下一个消息顺序号
```
**改进方向**  

****

一个针对于用户管理自身api_key的页面路由编制//仅仅这是个简单的计划，用于确认服务的功能而已

| 路由   | 负责的功能                                               | 是否受保护 |
|:-----|:----------------------------------------------------|:------|
| /... | 用户创建api的路由                                          | 是     |
| /... | 获取用户全部api                                           | 是     |
| /... | 用户更新api的路由                                          | 是     |
| /... | 用户删除单个api的路由路由                                      | 是     |
| /... | 前端根据那个用户api的名字获取对应api的路由的详细消息，毕竟这个功能可能在那个对话页面当中要被使用 | 是     |
| /... | 前端根据那个用户全部api的情况默认获取使用的api，例如下拉列表初始化时，就先调用这个临时顶着    | 是     |