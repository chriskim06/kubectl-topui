package config

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

const configPath = "~/.config/kubectl-topui/config.yml"

var (
	config          Config
	once            sync.Once
	defaultSelected = Color("13")
	defaultLimit    = Color("9")
	defaultUsage    = Color("10")
	defaultColors   = Colors{
		Selected: defaultSelected,
		CPULimit: defaultLimit,
		CPUUsage: defaultUsage,
		MemLimit: defaultLimit,
		MemUsage: defaultUsage,
	}
)

type Config struct {
	Theme Colors `json:"theme" yaml:"theme"`
}

type Color lipgloss.Color

func (c *Color) UnmarshalJSON(data []byte) error {
	*c = Color(lipgloss.Color(string(data)))
	return nil
}

type Colors struct {
	Selected Color `json:"selected" yaml:"selected"`
	CPULimit Color `json:"cpuLimit" yaml:"cpuLimit"`
	CPUUsage Color `json:"cpuUsage" yaml:"cpuUsage"`
	MemLimit Color `json:"memLimit" yaml:"memLimit"`
	MemUsage Color `json:"memUsage" yaml:"memUsage"`
}

func initConfig() {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.config/kubectl-topui/")
		viper.SetDefault("theme.selected", defaultSelected)
		viper.SetDefault("theme.cpuLimit", defaultLimit)
		viper.SetDefault("theme.cpuUsage", defaultUsage)
		viper.SetDefault("theme.memLimit", defaultLimit)
		viper.SetDefault("theme.memUsage", defaultUsage)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found use default
				config = Config{Theme: defaultColors}
				return
			}
		}
		var c Config
		if err := viper.Unmarshal(&c); err != nil {
			fmt.Println("unmarshal error", err)
			return
		}
		config = c
	})
}

func GetTheme() Colors {
	initConfig()
	return config.Theme
}
