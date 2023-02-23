package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

const configPath = "~/.config/kubectl-topui/config.yml"

var (
	config          Config
	once            sync.Once
	defaultSelected = ColorString("pink")
	defaultLimit    = ColorString("red")
	defaultUsage    = ColorString("darkcyan")
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

type ColorString string

func (c *ColorString) UnmarshalJSON(data []byte) error {
	s := string(data)
	if strings.Contains(s, "\"") {
		s = strings.Trim(s, "\"")
	}
	*c = ColorString(s)
	return nil
}

type Colors struct {
	Selected ColorString `json:"selected" yaml:"selected"`
	CPULimit ColorString `json:"cpuLimit" yaml:"cpuLimit"`
	CPUUsage ColorString `json:"cpuUsage" yaml:"cpuUsage"`
	MemLimit ColorString `json:"memLimit" yaml:"memLimit"`
	MemUsage ColorString `json:"memUsage" yaml:"memUsage"`
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
