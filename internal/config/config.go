package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	WatchDir              string `yaml:"watch_dir" json:"watchDir"`
	ArchiveDir            string `yaml:"archive_dir" json:"archiveDir"`
	StabilityDelaySeconds int    `yaml:"stability_delay_seconds" json:"stabilityDelaySeconds"`
	WorkerCount           int    `yaml:"worker_count" json:"workerCount"`
	AutoCleanupDays       int    `yaml:"auto_cleanup_days" json:"autoCleanupDays"`     // 0 = disabled
	KeepOriginal          bool   `yaml:"keep_original" json:"keepOriginal"`           // copy instead of move
	WebhookURL            string `yaml:"webhook_url" json:"webhookUrl"`               // URL called on errors
	NotificationSound     bool   `yaml:"notification_sound" json:"notificationSound"` // play sound on learning
	ExtraWatchDirs        []string `yaml:"extra_watch_dirs" json:"extraWatchDirs"`     // additional watch folders
	AllowedPrinters       []string `yaml:"allowed_printers" json:"allowedPrinters"`   // restrict printer list (empty = all)
}

func Load(path string) (Config, error) {
	cfg := Config{
		StabilityDelaySeconds: 2,
		WorkerCount:           2,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.StabilityDelaySeconds <= 0 {
		cfg.StabilityDelaySeconds = 2
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
