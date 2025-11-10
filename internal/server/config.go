package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"project/internal/domain/errors"
	"strconv"
)

// Config представляет конфигурацию сервера.
// Содержит настройки адреса, порта, строки подключения к БД, пути к миграциям и параметры HTTPS.
// HTTPS можно включить через флаг -s или переменную окружения ENABLE_HTTPS.
type Config struct {
	Addr        string // Адрес сервера
	Port        int    // Порт сервера
	DBStr       string // Строка подключения к базе данных
	MigratePath string // Путь к папке с миграциями
	EnableHTTPS bool   // Включить HTTPS (флаг -s или переменная окружения ENABLE_HTTPS)
	CertFile    string // Путь к файлу сертификата для HTTPS (переменная окружения CERT_FILE или флаг -cert)
	KeyFile     string // Путь к файлу приватного ключа для HTTPS (переменная окружения KEY_FILE или флаг -key)
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
	enableHTTPS = flag.Bool("s", false, "включить HTTPS")
	certFile    = flag.String("cert", "", "путь к файлу сертификата для HTTPS")
	keyFile     = flag.String("key", "", "путь к файлу приватного ключа для HTTPS")
	parsed      = false
)

// ReadConfig читает конфигурацию из различных источников.
// Приоритет источников: флаги командной строки > переменные окружения > JSON файл > значения по умолчанию.
// Возвращает указатель на структуру Config с загруженными настройками.
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
		EnableHTTPS: false,
		CertFile:    "",
		KeyFile:     "",
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

	if enableHTTPS := os.Getenv("ENABLE_HTTPS"); enableHTTPS != "" {
		if enableHTTPS == "true" || enableHTTPS == "1" || enableHTTPS == "yes" {
			cfg.EnableHTTPS = true
		}
	}

	if certFile := os.Getenv("CERT_FILE"); certFile != "" {
		cfg.CertFile = certFile
	}

	if keyFile := os.Getenv("KEY_FILE"); keyFile != "" {
		cfg.KeyFile = keyFile
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

	if *enableHTTPS {
		cfg.EnableHTTPS = true
	}

	if *certFile != "" {
		cfg.CertFile = *certFile
	}

	if *keyFile != "" {
		cfg.KeyFile = *keyFile
	}

	return cfg
}
