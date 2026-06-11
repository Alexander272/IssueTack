package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Environment string `yaml:"environment" env:"APP_ENV" env-default:"dev"`
		LogLevel    string `yaml:"log_level" env-default:"info"`
		LogSource   bool   `yaml:"log_source" env-default:"false"`
		// Redis        RedisConfig
		Postgres      PostgresConfig   `yaml:"postgres"`
		Auth          AuthConfig       `yaml:"auth"`
		Keycloak      KeycloakConfig   `yaml:"keycloak"`
		Http          HttpConfig       `yaml:"http"`
		ApiLimiter    LimiterConfig    `yaml:"api_limiter"`
		StaticLimiter LimiterConfig    `yaml:"static_limiter"`
		Casbin        CasbinConfig     `yaml:"casbin"`
		FileServer    FileServerConfig `yaml:"file_server"`
	}

	FileServerConfig struct {
		UploadDir string `yaml:"upload_dir" env:"UPLOAD_DIR" env-default:"./uploads"`
		MaxSize   int64  `yaml:"max_size" env:"UPLOAD_MAX_SIZE" env-default:"10485760"`
	}

	HttpConfig struct {
		Host               string        `yaml:"host" env:"HOST" env-default:"localhost"`
		Port               string        `yaml:"port" env:"PORT" env-default:"8080"`
		ReadTimeout        time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"10s"`
		WriteTimeout       time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s"`
		MaxHeaderMegabytes int           `yaml:"max_header_bytes" env-default:"1"`
		AllowedOrigins     []string      `yaml:"allowed_origins" env:"ALLOWED_ORIGINS"`
		WriteWait          time.Duration `yaml:"write_wait" env-default:"10s"`
		PongWait           time.Duration `yaml:"pong_wait" env-default:"60s"`
		PingPeriod         time.Duration `yaml:"ping_period" env-default:"54s"`
		MaxMessageSize     int64         `yaml:"max_message_size" env-default:"10240"`
	}

	// RedisConfig struct {
	// 	Host     string `yaml:"host" env:"REDIS_HOST"`
	// 	Port     string `yaml:"port" env:"REDIS_PORT"`
	// 	DB       int    `yaml:"db" env:"REDIS_DB"`
	// 	Password string `env:"REDIS_PASSWORD"`
	// }

	PostgresConfig struct {
		Host     string `yaml:"host" env:"POSTGRES_HOST"`
		Port     string `yaml:"port" env:"POSTGRES_PORT"`
		Username string `yaml:"username" env:"POSTGRES_NAME"`
		Password string `env:"POSTGRES_PASSWORD"`
		DbName   string `yaml:"db_name" env:"POSTGRES_DB"`
		SSLMode  string `yaml:"ssl_mode" env:"POSTGRES_SSL"`
	}

	AuthConfig struct {
		AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env-default:"10m"`
		RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env-default:"12h"`
		LimitAuthTTL    time.Duration `yaml:"limit_auth_ttl" env-default:"30m"`
		CountAttempt    int32         `yaml:"count_attempt" env-default:"5"`
		Secure          bool          `yaml:"secure" env-default:"false"`
		Domain          string        `yaml:"domain" env-default:"sealur.ru"`
		Key             string        `env:"JWT"`
	}

	KeycloakConfig struct {
		Url          string `yaml:"keycloak_url" env:"KEYCLOAK_URL"`
		ClientId     string `env:"KEYCLOAK_CLIENT_ID"`
		ClientSecret string `env:"KEYCLOAK_CLIENT_SECRET"`
		Realm        string `yaml:"keycloak_realm" env:"KEYCLOAK_REALM"`
		Root         string `env:"KEYCLOAK_ROOT"`
		RootPass     string `env:"KEYCLOAK_ROOT_PASS"`
	}

	LimiterConfig struct {
		RPS   int           `yaml:"rps" env:"RPS" env-default:"10"`
		Burst int           `yaml:"burst" env:"BURST" env-default:"20"`
		TTL   time.Duration `yaml:"ttl" env:"TTL" env-default:"5m"`
	}

	CasbinConfig struct {
		ModelPath     string `yaml:"model_path" env:"CASBIN_MODEL_PATH" env-default:"/configs/privacy.conf"`
		EnableWatcher bool   `yaml:"enable_watcher" env:"CASBIN_ENABLE_WATCHER" env-default:"false"`
	}
)

func Init(path string) (*Config, error) {
	var conf Config

	if err := cleanenv.ReadConfig(path, &conf); err != nil {
		return nil, fmt.Errorf("failed to read config file. error: %w", err)
	}

	// if err := cleanenv.ReadEnv(&conf); err != nil {
	// 	return nil, fmt.Errorf("failed to read env file. error: %w", err)
	// }

	return &conf, nil
}
