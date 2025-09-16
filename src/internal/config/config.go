package config

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Configuration struct {
	Logs     LogsSettings     `mapstructure:"logs"`
	App      Application      `mapstructure:"app"`
	Database Database         `mapstructure:"database"`
	Queue    QueueConfig      `mapstructure:"queue"`
	Redis    Redis            `mapstructure:"redis"`
	Security SecuritySettings `mapstructure:"security"`
	Server   ServerSettings   `mapstructure:"server"`
	Search   SearchConfig     `mapstructure:"search"`
	Cache    CacheConfig      `mapstructure:"cache"`
}

type LogsSettings struct {
	Level            string `mapstructure:"level"`
	Path             string `mapstructure:"log-path"`
	EnableJSONOutput bool   `mapstructure:"enable-json-output"`
}

type Application struct {
	Name     string `mapstructure:"name"`
	Timeout  int    `mapstructure:"timeout"`
	Version  string `mapstructure:"version"`
	HostLink string `mapstructure:"host-link"`
}

type Database struct {
	Url               string `mapstructure:"url"`
	DbName            string `mapstructure:"dbname"`
	UserCollection    string `mapstructure:"user-collection"`
	SessionCollection string `mapstructure:"session-collection"`
	Timeout           int    `mapstructure:"timeout"`
}

type SearchConfig struct {
	MinQueryLimit int `mapstructure:"min-query-limit"`
	MaxQueryLimit int `mapstructure:"min-query-limit"`
}

type QueueConfig struct {
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
}

type RabbitMQConfig struct {
	Url            string `mapstructure:"url"`
	Exchange       string `mapstructure:"exchange"`
	ExchangeType   string `mapstructure:"exchange-type"`
	EmailQueue     string `mapstructure:"email-queue"`
	PrefetchCount  int    `mapstructure:"prefetch-count"`
	ReconnectDelay int    `mapstructure:"reconnect-delay"`
	Timeout        int    `mapstructure:"timeout"`
	RoutingKey     string `mapstructure:"routing-key"`
	PrefetchSize   int    `mapstructure:"prefetch-size"`
	Global         bool   `mapstructure:"global"`
	Durable        bool   `mapstructure:"durable"`
	AutoDelete     bool   `mapstructure:"auto-delete"`
	Internal       bool   `mapstructure:"internal"`
	NoWait         bool   `mapstructure:"no-wait"`
	Exclusive      bool   `mapstructure:"exclusive"`
	AutoAck        bool   `mapstructure:"auto-ack"`
	NoLocal        bool   `mapstructure:"no-local"`
	Consumer       string `mapstructure:"consumer"`
}

type Redis struct {
	Url      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	Db       int    `mapstructure:"db"`
}

type SecuritySettings struct {
	JwtKey string `mapstructure:"jwt-key"`
}

type ServerSettings struct {
	Port         string `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	ReadTimeout  int    `mapstructure:"read-timeout"`
	WriteTimeout int    `mapstructure:"write-timeout"`
	IdleTimeout  int    `mapstructure:"idle-timeout"`
}

type CacheConfig struct {
	ExpirationMinutes         int    `mapstructure:"expiration-minutes"`
	ExtendedExpirationMinutes int    `mapstructure:"extended-expiration-minutes"`
	SessionExpirationMinutes  int    `mapstructure:"session-expiration-minutes"`
	UserStatKey               string `mapstructure:"user-stat-key"`
	UsetStatExpirationMinutes int    `mapstructure:"user-stat-expiration-minutes"`
}

func Load() *Configuration {
	cfg := read()
	logrus.Info("Configuration loaded")

	// Override with environment variables
	mongoUri := os.Getenv("MONGODB_URL")
	if mongoUri != "" {
		cfg.Database.Url = mongoUri
	}

	dbName := os.Getenv("DB_NAME")
	if dbName != "" {
		cfg.Database.DbName = dbName
	}

	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl != "" {
		cfg.Redis.Url = redisUrl
	}

	redisDB := os.Getenv("REDIS_DB")
	if redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			cfg.Redis.Db = db
		}
	}

	rabbitmqUrl := os.Getenv("RABBITMQ_URL")
	if rabbitmqUrl != "" {
		cfg.Queue.RabbitMQ.Url = rabbitmqUrl
	}

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey != "" {
		cfg.Security.JwtKey = jwtKey
	}

	return cfg
}

func read() *Configuration {
	viper.SetConfigFile("src/internal/config/cfg.yml")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	var config Configuration

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Panic("Error reading config file, %s", err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		logrus.Panic("Error unmarshalling config file, %s", err)
	}

	return &config
}
