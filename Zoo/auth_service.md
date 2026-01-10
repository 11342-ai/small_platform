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
| /api/register            | 注册的路由           | 否     |
| /api/login               | 登录的路由           | 否     |
| /api/logout              | 推出登录的路由         | 否     |
| /api/auth/send-code      | 验证时发送验证码请求的路由   | 否     |
| /api/auth/verify-code    | 验证时填入验证码后路由     | 否     |
| /api/auth/reset-password | 忘记密码的哪个路由       | 否     |
| /api/profile             | 查看个人资料的路由       | 是     |
| /api/update-password     | 更新个人密码的路由       | 是     |
| /api/me                  | 为前端提供更友好的用户信息端点 | 是     |

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

********************************

**用户个人API配置方面地接口**  
&emsp;&emsp;这个主要是针对于那个用户个人api地增删改查而已

```
// UserAPIServiceInterface 用户API配置服务接口
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
CreateAPI ====》根据用户ID和API配置创建一个新的API记录
GetAPIByID ====》根据API ID获取具体的API配置
GetAPIByName ====》通过用户ID和API名称获取API的详细信息
GetAPIByModelName ====》通过用户ID和模型名称获取对应的API配置（聊天时选择模型用）
GetUserAPIs ====》获取用户的所有API配置列表
UpdateAPI ====》更新API配置信息
DeleteAPI ====》删除指定的API配置
TestAPIConnection ====》测试API连接（简单实现，始终返回true）
GetFirstAvailableAPI ====》获取用户第一个可用的API配置（用于下拉列表默认值）
```

一个针对于用户管理自身api_key的页面路由编制//实际实现的路由

| 路由                          | 负责的功能                                       | 是否受保护 |
|:----------------------------|:--------------------------------------------|:------|
| /api/user/apis              | 创建新的API配置（POST）                           | 是     |
| /api/user/apis              | 获取用户的所有API配置列表（GET）                     | 是     |
| /api/user/apis/first        | 获取用户第一个可用的API配置（用于下拉列表默认值）（GET）   | 是     |
| /api/user/apis/:name        | 根据API名称获取具体的API配置（GET）                  | 是     |
| /api/user/apis/:id          | 更新API配置（PUT）                               | 是     |
| /api/user/apis/:id          | 删除API配置（DELETE）                            | 是     |

********************************


