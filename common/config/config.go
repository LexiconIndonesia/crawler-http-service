package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// get secret from file, or fallback to env
func getSecret(key string) string {
	value, exists := os.LookupEnv(key + "_FILE")
	if exists {
		// trim space and newline
		secret, err := os.ReadFile(value)
		if err == nil {
			return strings.TrimSpace(string(secret))
		}
	}

	return getEnv(key, "")
}

func loadEnvString(key string, result *string) {
	s, ok := os.LookupEnv(key)

	if !ok {
		return
	}
	*result = s
}

func loadEnvUint(key string, result *uint) {
	s, ok := os.LookupEnv(key)

	if !ok {
		return
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	*result = uint(n)
}

/* Configuration */

/* PgSQL Configuration */
type pgSqlConfig struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Database string `json:"database"`
	SslMode  string `json:"ssl_mode"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func (p pgSqlConfig) ConnStr() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=%s", p.Host, p.Port, p.User, p.Password, p.Database, p.SslMode)
}

func defaultPgSql() pgSqlConfig {
	return pgSqlConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "database",
		User:     "",
		Password: "",
		SslMode:  "disable",
	}
}

func (p *pgSqlConfig) loadFromEnv() {
	loadEnvString("POSTGRES_HOST", &p.Host)
	loadEnvUint("POSTGRES_PORT", &p.Port)
	loadEnvString("POSTGRES_DB_NAME", &p.Database)
	loadEnvString("POSTGRES_SSLMODE", &p.SslMode)
	loadEnvString("POSTGRES_USERNAME", &p.User)
	p.Password = getSecret("POSTGRES_PASSWORD")
}

/* Listen Configuration */

type listenConfig struct {
	Host string `json:"host"`
	Port uint   `json:"port"`
}

func (l listenConfig) Addr() string {
	return fmt.Sprintf("%s:%d", l.Host, l.Port)
}

func defaultListenConfig() listenConfig {
	return listenConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}
}

func (l *listenConfig) loadFromEnv() {
	loadEnvString("LISTEN_HOST", &l.Host)
	loadEnvUint("LISTEN_PORT", &l.Port)
}

type hostConfig struct {
	Host string `json:"host"`
}

func (h *hostConfig) loadFromEnv() {
	loadEnvString("HOST", &h.Host)
}

func defaultHostConfig() hostConfig {
	return hostConfig{
		Host: "localhost",
	}
}

type natsConfig struct {
	Host             string
	Port             uint
	Username         string
	Password         string
	JetStreamEnabled bool
	PortMonitoring   uint
}

func (c *natsConfig) loadFromEnv() {
	c.Host = getEnv("NATS_HOST", "localhost")

	// Load port with default 4222
	if portStr := getEnv("NATS_PORT", "4222"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			c.Port = uint(port)
		} else {
			c.Port = 4222
		}
	} else {
		c.Port = 4222
	}

	c.Username = getSecret("NATS_USER")
	c.Password = getSecret("NATS_PASSWORD")

	// Load JetStream enabled flag
	if jsEnabled := getEnv("NATS_JETSTREAM_ENABLED", "true"); jsEnabled == "true" {
		c.JetStreamEnabled = true
	} else {
		c.JetStreamEnabled = false
	}

	// Load monitoring port
	if portMonitorStr := getEnv("NATS_PORT_MONITORING", "8222"); portMonitorStr != "" {
		if portMonitor, err := strconv.Atoi(portMonitorStr); err == nil {
			c.PortMonitoring = uint(portMonitor)
		} else {
			c.PortMonitoring = 8222
		}
	} else {
		c.PortMonitoring = 8222
	}
}

func (c *natsConfig) URL() string {
	return fmt.Sprintf("nats://%s:%d", c.Host, c.Port)
}

func defaultNatsConfig() natsConfig {
	return natsConfig{
		Host:             "localhost",
		Port:             4222,
		Username:         "",
		Password:         "",
		JetStreamEnabled: true,
		PortMonitoring:   8222,
	}
}

type securityConfig struct {
	BackendApiKey string
	ServerSalt    string
}

func (s *securityConfig) loadFromEnv() {
	s.BackendApiKey = getEnv("BACKEND_API_KEY", "")
	s.ServerSalt = getEnv("SERVER_SALT", "")
}

func defaultSecurityConfig() securityConfig {
	return securityConfig{
		BackendApiKey: "",
		ServerSalt:    "",
	}
}

type redisConfig struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func (r *redisConfig) loadFromEnv() {
	loadEnvString("REDIS_HOST", &r.Host)
	loadEnvUint("REDIS_PORT", &r.Port)
	r.Password = getSecret("REDIS_PASSWORD")

	// Load DB number with a default of 0
	if dbStr := getEnv("REDIS_DB", "0"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			r.DB = db
		}
	}
	log.Info().Interface("redis", r).Msg("Redis config loaded")
}

func defaultRedisConfig() redisConfig {
	return redisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}
}

type GCSConfig struct {
	ProjectID       string
	CredentialsFile string
	Bucket          string
}

func (g *GCSConfig) loadFromEnv() {
	g.ProjectID = getEnv("GCS_PROJECT_ID", "")
	g.CredentialsFile = getEnv("GCS_CREDENTIALS_FILE", "")
	g.Bucket = getEnv("GCS_STORAGE_BUCKET", "")
}

func defaultGcsConfig() GCSConfig {
	return GCSConfig{
		ProjectID:       "",
		CredentialsFile: "",
		Bucket:          "",
	}
}

type Config struct {
	Host     hostConfig
	Listen   listenConfig
	PgSql    pgSqlConfig
	Security securityConfig
	Nats     natsConfig
	Redis    redisConfig
	GCS      GCSConfig
}

func (c *Config) LoadFromEnv() {
	c.Host.loadFromEnv()
	c.Listen.loadFromEnv()
	c.PgSql.loadFromEnv()
	c.Security.loadFromEnv()
	c.Nats.loadFromEnv()
	c.Redis.loadFromEnv()
	c.GCS.loadFromEnv()
}

func DefaultConfig() Config {
	return Config{
		Host:     defaultHostConfig(),
		Listen:   defaultListenConfig(),
		PgSql:    defaultPgSql(),
		Security: defaultSecurityConfig(),
		Nats:     defaultNatsConfig(),
		Redis:    defaultRedisConfig(),
		GCS:      defaultGcsConfig(),
	}
}
