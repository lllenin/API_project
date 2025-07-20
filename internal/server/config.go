package server

import (
	"flag"
	"fmt"
	"os"
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
	parsed      = false
)

func ReadConfig() *Config {
	if !parsed {
		flag.Parse()
		parsed = true
	}
	cfg := &Config{
		Addr:        *addr,
		Port:        *port,
		DBStr:       *dbstr,
		MigratePath: *migratePath,
	}
	if *dbDsn != "" {
		cfg.DBStr = *dbDsn
	} else if *dbstr == defaultDBStr {
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
