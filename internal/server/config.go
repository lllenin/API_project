package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"project/internal/domain/errors"
	"strconv"
)

type Config struct {
	Addr        string
	Port        int
	DBStr       string
	MigratePath string
}

const (
	defaultAddr        = "0.0.0.0"
	defaultPort        = 8080
	defaultDBStr       = "postgresql://shouldbeinVaultuser:shouldbeinVaultpassword@db:5432/tasks?sslmode=disable"
	defaultMigratePath = "migrations"
)

var (
	addr        = flag.String("addr", defaultAddr, "адрес сервера (по умолчанию 0.0.0.0)")
	port        = flag.Int("port", defaultPort, "порт сервера (по умолчанию 8080)")
	dbstr       = flag.String("dbstr", defaultDBStr, "строка подключения к БД (по умолчанию стандартная)")
	dbDsn       = flag.String("dbdsn", "", "DSN для подключения к базе данных (приоритетнее dbstr)")
	migratePath = flag.String("migratepath", defaultMigratePath, "путь к папке с миграциями")
	configFile  = flag.String("c", "", "путь к файлу конфигурации JSON")
	parsed      = false
)

func ReadConfig() *Config {
	if !parsed {
		flag.Parse()
		parsed = true
	}

	cfg := &Config{
		Addr:        defaultAddr,
		Port:        defaultPort,
		DBStr:       defaultDBStr,
		MigratePath: defaultMigratePath,
	}

	jsonConfig := loadJSONConfig()
	if jsonConfig != nil {
		cfg = jsonConfig
	}

	cfg = applyEnvOverrides(cfg)
	cfg = applyFlagOverrides(cfg)

	return cfg
}

func loadJSONConfig() *Config {
	configPath := *configFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	if configPath == "" {
		fmt.Printf("JSON конфигурация: не указан путь к файлу\n")
		return nil
	}

	fmt.Printf("Загрузка JSON конфигурации из: %s\n", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Warning: %s %s: %v\n", errors.ErrConfigFileReadFailed.Error(), configPath, err)
		return nil
	}

	var jsonConfig Config
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		fmt.Printf("Warning: %s: %v\n", errors.ErrConfigParseFailed.Error(), err)
		return nil
	}

	fmt.Printf("JSON конфигурация успешно загружена из: %s\n", configPath)
	return &jsonConfig
}

func applyEnvOverrides(cfg *Config) *Config {
	if addr := os.Getenv("ADDR"); addr != "" {
		cfg.Addr = addr
	}
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err != nil {
			fmt.Printf("Warning: %s в переменной окружения PORT: %s\n", errors.ErrConfigInvalidFormat.Error(), port)
		} else if p < 1 || p > 65535 {
			fmt.Printf("Warning: %s - порт должен быть от 1 до 65535: %d\n", errors.ErrConfigInvalidFormat.Error(), p)
		} else {
			cfg.Port = p
		}
	}
	if dbStr := os.Getenv("DB_STR"); dbStr != "" {
		cfg.DBStr = dbStr
	}
	if migratePath := os.Getenv("MIGRATE_PATH"); migratePath != "" {
		cfg.MigratePath = migratePath
	}

	if cfg.DBStr == defaultDBStr {
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		if dbUser != "" && dbPassword != "" && dbName != "" && dbHost != "" && dbPort != "" {
			cfg.DBStr = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
		}
	}

	return cfg
}

func applyFlagOverrides(cfg *Config) *Config {
	cfg.Addr = *addr
	cfg.Port = *port
	cfg.MigratePath = *migratePath

	if *dbDsn != "" {
		cfg.DBStr = *dbDsn
	} else {
		cfg.DBStr = *dbstr
	}

	return cfg
}
