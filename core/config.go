package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ServerConfig struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	KeyPath    string `json:"keyPath"`
	RemotePath string `json:"remotePath"`
}

type ShellConfig struct {
	MenuTitle       string `json:"menuTitle"`       // Default: "Rsync"
	Position        string `json:"position"`        // "Default", "Top", "Bottom"
	ShowIcons       bool   `json:"showIcons"`       // Default: true
	CustomIconPath  string `json:"customIconPath"`  // Optional custom icon
	UseSeparators   bool   `json:"useSeparators"`   // Default: true
	UseAccelerators bool   `json:"useAccelerators"` // Default: true (&Copy to, &Pull from)
	SubmenuDepth    int    `json:"submenuDepth"`    // Default: 1 (1-level flyout)
}

type CustomThemeConfig struct {
	Name         string `json:"name,omitempty"`
	CanvasBg     string `json:"canvasBg,omitempty"`     // Hex, e.g. "#1a2332"
	SurfaceBg    string `json:"surfaceBg,omitempty"`    // Hex, e.g. "#c0c0c0"
	ButtonBg     string `json:"buttonBg,omitempty"`     // Hex, e.g. "#d4d0c8"
	ButtonText   string `json:"buttonText,omitempty"`   // Hex, e.g. "#000000"
	AccentColor  string `json:"accentColor,omitempty"`  // Hex, e.g. "#0284c7"
	AccentText   string `json:"accentText,omitempty"`   // Hex, e.g. "#ffffff"
	BorderColor  string `json:"borderColor,omitempty"`  // Hex, e.g. "#0f172a"
	TextColor    string `json:"textColor,omitempty"`    // Hex, e.g. "#000000"
	CornerRadius int    `json:"cornerRadius,omitempty"` // 0, 4, 8, 14, 20
	ShadowDepth  int    `json:"shadowDepth,omitempty"`  // 0, 1, 2, 4, 6
}

type AppConfig struct {
	ActiveTheme      string             `json:"activeTheme"`
	CornerRadius     int                `json:"cornerRadius"`
	ShadowDepth      int                `json:"shadowDepth"` // 0=flat, 1=1px, 2=2px, 4=4px, 6=6px
	PreferredIndexer EngineType         `json:"preferredIndexer"`
	NetworkTimeoutMs int                `json:"networkTimeoutMs"`
	RsyncBinaryPath  string             `json:"rsyncBinaryPath"`
	ShowHiddenFiles  bool               `json:"showHiddenFiles"`
	Shell            ShellConfig        `json:"shell"`
	Servers          []*ServerConfig    `json:"servers"`
	CustomTheme      *CustomThemeConfig `json:"customTheme,omitempty"`
}

func DefaultCustomTheme() *CustomThemeConfig {
	return &CustomThemeConfig{
		Name:         "Custom Theme",
		CanvasBg:     "1a2332",
		SurfaceBg:    "c0c0c0",
		ButtonBg:     "d4d0c8",
		ButtonText:   "000000",
		AccentColor:  "0284c7",
		AccentText:   "ffffff",
		BorderColor:  "0f172a",
		TextColor:    "000000",
		CornerRadius: 8,
		ShadowDepth:  2,
	}
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		ActiveTheme:      "y2k-candy",
		CornerRadius:     8,
		ShadowDepth:      2,
		CustomTheme:      DefaultCustomTheme(),
		PreferredIndexer: EngineUSN,
		NetworkTimeoutMs: 500,
		RsyncBinaryPath:  "rsync",
		ShowHiddenFiles:  false,
		Shell: ShellConfig{
			MenuTitle:       "Rsync",
			Position:        "Default",
			ShowIcons:       true,
			CustomIconPath:  "",
			UseSeparators:   true,
			UseAccelerators: true,
			SubmenuDepth:    1,
		},
		Servers: []*ServerConfig{
			{
				ID:         "alpha",
				Name:       "Alpha Server",
				Host:       "192.168.1.100",
				Port:       22,
				User:       "admin",
				RemotePath: "/data/backup",
			},
		},
	}
}

func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = SaveConfig(path, cfg)
			return cfg, nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(path string, cfg *AppConfig) error {
	dir := filepath.Dir(path)
	if dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
