package config

import (
	"flag"
	"fmt"
	"github.com/blaize9/vuze-tools/utils"
	"github.com/jinzhu/configor"
	"path/filepath"
	"strings"
	"sync"
)

var (
	DefaultConfigPath = "config/default_config.yml"
	ConfigPath        = "config/config.yml"
)

var config *Config
var once sync.Once

type Config struct {
	Port                        int    `json:"port" yaml:"port"`
	Version                     string `json:"build_version" yaml:"build_version"`
	LockFilename                string `json:"lock_filename" yaml:"lock_filename"`
	AzureusDirectory            string `json:"azureus_directory" yaml:"azureus_directory"`
	AzureusTorrentsDirectory    string `json:"azureus_torrents_directory" yaml:"azureus_torrents_directory"`
	AzureusDownloadsConfig      string `json:"azureus_downloads_config" yaml:"azureus_downloads_config,omitempty"`
	AzureusRecoverTempDirectory string `json:"azureus_recover_temp_directory" yaml:"azureus_recover_temp_directory,omitempty"`

	SimpleRecoverWorkers      int           `json:"simple_recovery_workers" yaml:"simple_recovery_workers,omitempty"`
	AdvancedRecoverMaxWorkers int           `json:"advanced_recovery_max_workers" yaml:"advanced_recovery_max_workers,omitempty"`
	AzureusBackupDirectories  AzDirectories `yaml:"azureus_backup_directories,flow,omitempty"`

	Environment string    `json:"environment" yaml:"environment,omitempty"`
	Log         LogConfig `yaml:"log,flow,omitempty"`
}

type LogConfig struct {
	AccessLogFilePath      string `yaml:"access_log_filepath,omitempty"`
	AccessLogFileExtension string `yaml:"access_log_fileextension,omitempty"`
	AccessLogMaxSize       int    `yaml:"access_log_max_size,omitempty"`
	AccessLogMaxBackups    int    `yaml:"access_log_max_backups,omitempty"`
	AccessLogMaxAge        int    `yaml:"access_log_max_age,omitempty"`
	ErrorLogFilePath       string `yaml:"error_log_filepath,omitempty"`
	ErrorLogFileExtension  string `yaml:"error_log_fileextension,omitempty"`
	ErrorLogMaxSize        int    `yaml:"error_log_max_size,omitempty"`
	ErrorLogMaxBackups     int    `yaml:"error_log_max_backups,omitempty"`
	ErrorLogMaxAge         int    `yaml:"error_log_max_age,omitempty"`
}

type AzDirectories []struct {
	Directory string
}

func init() {
	configor.Load(Get(), ConfigPath, DefaultConfigPath)
}

func Get() *Config {
	once.Do(func() {
		config = &Config{}
	})
	return config
}

func GetAzDownloadsConfig() string {
	if filepath.Dir(Get().AzureusDownloadsConfig) == "." {
		//file
		return filepath.Join(Get().AzureusDirectory, Get().AzureusDownloadsConfig)
	} else {
		//filepath
		return Get().AzureusDownloadsConfig
	}
}

func GetAzTorrentsPath() string {
	var path string
	if Get().AzureusTorrentsDirectory == "" {
		path = filepath.Join(Get().AzureusDirectory, "torrents")
	} else {
		path = filepath.Join(Get().AzureusDirectory, Get().AzureusTorrentsDirectory)
	}
	return path

}

func GetAzActivePath() string {
	return filepath.Join(Get().AzureusDirectory, "active")

}

func GetAzRecoverPath() string {
	var path string
	if Get().AzureusRecoverTempDirectory == "" || Get().AzureusRecoverTempDirectory == "azureus-recover" {
		path = filepath.Join(Get().AzureusDirectory, "../azureus-recover")
	} else {
		path = filepath.Join(Get().AzureusDirectory, Get().AzureusRecoverTempDirectory)
	}

	return path
}

func BindFLags() func() {
	var backupdirs string
	flag.StringVar(&Get().Environment, "env", Get().Environment, "Environment [DEV,PROD,PROD-STDOUT,PROD-JSON]")
	flag.StringVar(&Get().AzureusDirectory, "azdir", Get().AzureusDirectory, "Directory that contains Azureus storage")
	flag.StringVar(&Get().AzureusDownloadsConfig, "azconfig", Get().AzureusDownloadsConfig, "File or FilePath to the downloads.config")
	flag.Parse()
	if backupdirs != "" {
		for _, dir := range strings.Split(backupdirs, ",") {
			dir = strings.TrimSpace(dir)
			if utils.DirExists(dir) {
				Get().AzureusBackupDirectories = append(Get().AzureusBackupDirectories, struct{ Directory string }{Directory: dir})
			} else {
				fmt.Println("Backup directory %s does not exist!", dir)
			}
		}
	}
	return func() {
		configor.Load(Get(), ConfigPath, DefaultConfigPath)
	}
}
