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
	defaultSelected = 13
	defaultLimit    = 9
	defaultUsage    = 10
)

type Config struct {
	Theme Colors `json:"theme" yaml:"theme"`
}

type Colors struct {
	Selected int `json:"selected" yaml:"selected"`
	CPULimit int `json:"cpuLimit" yaml:"cpuLimit"`
	CPUUsage int `json:"cpuUsage" yaml:"cpuUsage"`
	MemLimit int `json:"memLimit" yaml:"memLimit"`
	MemUsage int `json:"memUsage" yaml:"memUsage"`
	Axis     int `json:"axis" yaml:"axis"`
	Labels   int `json:"labels" yaml:"labels"`
}

func initConfig() {
	once.Do(func() {
		defaultColor := 231
		if !lipgloss.HasDarkBackground() {
			defaultColor = 0
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.config/kubectl-topui/")
		viper.SetDefault("theme.selected", defaultSelected)
		viper.SetDefault("theme.cpuLimit", defaultLimit)
		viper.SetDefault("theme.cpuUsage", defaultUsage)
		viper.SetDefault("theme.memLimit", defaultLimit)
		viper.SetDefault("theme.memUsage", defaultUsage)
		viper.SetDefault("theme.axis", defaultColor)
		viper.SetDefault("theme.labels", defaultColor)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found use default
				config = Config{Theme: Colors{
					Selected: defaultSelected,
					CPULimit: defaultLimit,
					CPUUsage: defaultUsage,
					MemLimit: defaultLimit,
					MemUsage: defaultUsage,
					Axis:     defaultColor,
					Labels:   defaultColor,
				}}
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
