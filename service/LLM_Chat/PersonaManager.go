package LLM_Chat

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"sync"
)

type PersonaManagerInterface interface {
	GetPersonaContent(personaName string) string
	GetAvailablePersonas() []string
	SetDefaultPersona(personaName string)
	GetDefaultPersona() string
}

// PersonaConfig 人格配置
type PersonaConfig struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}

// PersonaConfigs 人格配置列表
type PersonaConfigs struct {
	Personas []PersonaConfig `yaml:"personas"`
}

type PersonaManager struct {
	configs        *PersonaConfigs
	defaultPersona string
	mu             sync.RWMutex
}

var GlobalPersonaManager PersonaManagerInterface

func NewPersonaManager(configs *PersonaConfigs) (PersonaManagerInterface, error) {
	if configs == nil {
		return nil, errors.New("人格配置不能为空")
	}

	if len(configs.Personas) == 0 {
		return nil, errors.New("至少需要配置一个人格")
	}

	if configs.Personas[0].Name == "" {
		return nil, errors.New("默认人格名称不能为空")
	}

	service := &PersonaManager{
		configs:        configs,
		defaultPersona: configs.Personas[0].Name,
	}
	GlobalPersonaManager = service
	return service, nil
}

// LoadPersonaConfigs 从YAML文件加载人格配置
func LoadPersonaConfigs(configPath string) (*PersonaConfigs, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var configs PersonaConfigs
	err = yaml.Unmarshal(data, &configs)
	if err != nil {
		return nil, err
	}

	log.Printf("加载了 %d 个人格配置", len(configs.Personas))
	return &configs, nil
}

// GetPersonaContent 根据人格名称获取内容
func (p *PersonaConfigs) GetPersonaContent(name string) string {
	for _, persona := range p.Personas {
		if persona.Name == name {
			return persona.Content
		}
	}
	return "" // 返回空表示使用默认
}

// GetPersonaNames 获取所有可用的人格名称
func (p *PersonaConfigs) GetPersonaNames() []string {
	names := make([]string, len(p.Personas))
	for i, persona := range p.Personas {
		names[i] = persona.Name
	}
	return names
}

// GetPersonaContent 根据人格名称获取内容
func (pm *PersonaManager) GetPersonaContent(personaName string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 如果传空或者默认，使用默认人格
	if personaName == "" || personaName == "default" {
		personaName = pm.defaultPersona
	}

	return pm.configs.GetPersonaContent(personaName)
}

// GetAvailablePersonas 获取所有可用的人格名称
func (pm *PersonaManager) GetAvailablePersonas() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.configs.GetPersonaNames()
}

// SetDefaultPersona 设置默认人格
func (pm *PersonaManager) SetDefaultPersona(personaName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 验证人格是否存在
	content := pm.configs.GetPersonaContent(personaName)
	if content == "" {
		return // 人格不存在，不设置
	}

	pm.defaultPersona = personaName
}

// GetDefaultPersona 获取默认人格
func (pm *PersonaManager) GetDefaultPersona() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.defaultPersona
}
