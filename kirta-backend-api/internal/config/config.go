package config

import (
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type PostgresPoolConfig struct {
	MaxConnections       int32         `yaml:"max_connections"`
	MinConnections       int32         `yaml:"min_connections"`
	MaxConnectionLifetime time.Duration `yaml:"max_connection_lifetime"`
	MaxConnIdleTime      time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod    time.Duration `yaml:"health_check_period"`
	ConnectTimeout       time.Duration `yaml:"connect_timeout"`
}

type PostgresConfig struct {
	Host       string             `yaml:"host"`
	Port       int                `yaml:"port"`
	Database   string             `yaml:"database"`
	SSL        string             `yaml:"ssl"`
	Username   string             `yaml:"username"`
	Password   string             `yaml:"password"`
	PoolConfig PostgresPoolConfig `yaml:"pool_config"`
}

type MinioCredentials struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type MinioConfig struct {
	Endpoint    string           `yaml:"endpoint"`
	UseSSL      bool             `yaml:"use_ssl"`
	Credentials MinioCredentials `yaml:"credentials"`
}

type LoggerConfig struct {
	Level        string `yaml:"level"`
	EnableCaller bool   `yaml:"enable_caller"`
	LogFile      string `yaml:"log_file"`
}

type HTTPConfig struct {
	Address           string        `yaml:"address"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes    int           `yaml:"max_header_bytes"`
}

type Config struct {
	Postgres PostgresConfig `yaml:"postgres"`
	Minio    MinioConfig    `yaml:"minio"`
	Logger   LoggerConfig   `yaml:"logger"`
	Http     HTTPConfig     `yaml:"http"`
	App      App            `yaml:"app"`
}

type App struct {
	GrypePath              string        `yaml:"grype_path"`
	SyftPath               string        `yaml:"syft_path"`
	ScaPath                string        `yaml:"sca_path"`
	GraphPath              string        `yaml:"graph_path"`
	BucketName             string        `yaml:"bucket_name"`
	OpenRouterAPIKey       string        `yaml:"openrouter_api_key"`
	OpenRouterModel        string        `yaml:"openrouter_model"`
	OpenRouterBaseURL      string        `yaml:"openrouter_base_url"`
	OpenRouterTimeout      time.Duration `yaml:"openrouter_timeout"`
	OpenRouterCallMapFiles int           `yaml:"openrouter_callmap_max_files"`
	OpenRouterCallMapCalls int           `yaml:"openrouter_callmap_max_calls"`
}

func New(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.App.OpenRouterBaseURL == "" {
		cfg.App.OpenRouterBaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.App.OpenRouterTimeout <= 0 {
		cfg.App.OpenRouterTimeout = 20 * time.Second
	}
	if cfg.App.OpenRouterCallMapFiles <= 0 {
		cfg.App.OpenRouterCallMapFiles = 20
	}
	if cfg.App.OpenRouterCallMapCalls <= 0 {
		cfg.App.OpenRouterCallMapCalls = 200
	}
	if cfg.Postgres.Port == 0 {
		cfg.Postgres.Port = 5432
	}
	if cfg.Postgres.SSL == "" {
		cfg.Postgres.SSL = "disable"
	}
	if cfg.Http.Address == "" {
		cfg.Http.Address = "0.0.0.0:8080"
	}
}
