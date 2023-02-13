package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"sigs.k8s.io/yaml"
)

const configPath = "~/.config/kubectl-ptop/config.yml"

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

func (c ColorString) Color() tcell.Color {
	return tcell.GetColor(string(c))
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
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		var f []byte
		f, err = ioutil.ReadFile(filepath.Join(home, ".config", "kubectl-ptop", "config.yml"))
		if err != nil {
			panic(err)
		}
		var c Config
		if err := yaml.Unmarshal(f, &c); err != nil {
			panic(err)
		}
		if c.Theme.Selected == "" {
			c.Theme.Selected = defaultSelected
		}
		if c.Theme.CPULimit == "" {
			c.Theme.CPULimit = defaultLimit
		}
		if c.Theme.CPUUsage == "" {
			c.Theme.CPUUsage = defaultUsage
		}
		if c.Theme.MemLimit == "" {
			c.Theme.MemLimit = defaultLimit
		}
		if c.Theme.MemUsage == "" {
			c.Theme.MemUsage = defaultUsage
		}
		config = c
	})
}

func GetTheme() Colors {
	initConfig()
	return config.Theme
}
