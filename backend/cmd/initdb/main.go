package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/internal/bootstrap"
	"github.com/example/gue/backend/internal/seeder"
	"github.com/example/gue/backend/pkg/db"
	"github.com/example/gue/backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	maxDBRetries = 30
	retryDelay   = 2 * time.Second
)

type initDBOptions struct {
	fresh                bool
	allowProductionFresh bool
}

func main() {
	opts := parseInitDBOptions()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	log := logger.New()
	migrationsSource, err := resolveMigrationSource()
	if err != nil {
		log.Error("failed to resolve migration source", "error", err.Error())
		os.Exit(1)
	}

	if _, err := waitForMySQLServer(cfg.Database, log); err != nil {
		log.Error("database server is not ready", "error", err.Error())
		os.Exit(1)
	}

	if opts.fresh {
		if strings.EqualFold(cfg.AppEnv, "production") && !opts.allowProductionFresh {
			log.Error("refusing fresh reset in production", "hint", "use --allow-production-fresh or INITDB_ALLOW_PRODUCTION_FRESH=true")
			os.Exit(1)
		}

		log.Warn("fresh mode enabled: dropping and recreating database", "database", cfg.Database.Name, "app_env", cfg.AppEnv)
		if err := resetDatabase(context.Background(), cfg.Database); err != nil {
			log.Error("failed to reset database", "error", err.Error())
			os.Exit(1)
		}
	} else if err := ensureDatabaseExists(context.Background(), cfg.Database); err != nil {
		log.Error("failed to ensure database exists", "error", err.Error())
		os.Exit(1)
	}

	sqlDB, err := waitForDatabase(cfg, log)
	if err != nil {
		log.Error("database is not ready", "error", err.Error())
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := runMigrations(cfg, migrationsSource); err != nil {
		log.Error("failed to run migrations", "error", err.Error())
		os.Exit(1)
	}
	log.Info("migrations completed")

	gormDB, err := db.NewGormMySQL(cfg.Database)
	if err != nil {
		log.Error("failed to initialize gorm mysql", "error", err.Error())
		os.Exit(1)
	}
	if err := seeder.SeedPayments(context.Background(), gormDB); err != nil {
		log.Error("failed to seed payments", "error", err.Error())
		os.Exit(1)
	}
	log.Info("payments seeded")

	devInput := bootstrap.DevUserInput{
		Name:     envOrDefault("BOOTSTRAP_DEV_NAME", "Developer"),
		Email:    strings.TrimSpace(os.Getenv("BOOTSTRAP_DEV_EMAIL")),
		Password: strings.TrimSpace(os.Getenv("BOOTSTRAP_DEV_PASSWORD")),
	}
	if err := bootstrap.EnsureSingleDevUser(context.Background(), sqlDB, devInput); err != nil {
		log.Error("failed to ensure single dev user", "error", err.Error())
		os.Exit(1)
	}
	log.Info("single dev user ensured", "email", devInput.Email)
}

func parseInitDBOptions() initDBOptions {
	freshFlag := flag.Bool("fresh", false, "drop and recreate database before applying migrations")
	allowProductionFreshFlag := flag.Bool("allow-production-fresh", false, "allow --fresh when APP_ENV=production")
	flag.Parse()

	return initDBOptions{
		fresh:                *freshFlag || envBool("INITDB_FRESH"),
		allowProductionFresh: *allowProductionFreshFlag || envBool("INITDB_ALLOW_PRODUCTION_FRESH"),
	}
}

func envBool(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func waitForDatabase(cfg config.Config, log *slog.Logger) (*sql.DB, error) {
	passwordCandidates := buildPasswordCandidates(cfg.Database.Password)
	var lastErr error
	for attempt := 1; attempt <= maxDBRetries; attempt++ {
		for _, candidate := range passwordCandidates {
			sqlDB, err := openDatabaseWithPassword(cfg.Database, candidate)
			if err != nil {
				lastErr = err
				continue
			}

			if candidate != cfg.Database.Password && strings.EqualFold(cfg.Database.User, "root") {
				log.Warn("database authenticated with fallback password, synchronizing configured password")
				if syncErr := syncRootPassword(context.Background(), sqlDB, cfg.Database.Password); syncErr != nil {
					_ = sqlDB.Close()
					lastErr = syncErr
					continue
				}
				_ = sqlDB.Close()

				sqlDB, err = openDatabaseWithPassword(cfg.Database, cfg.Database.Password)
				if err != nil {
					lastErr = fmt.Errorf("reconnect after password synchronization: %w", err)
					continue
				}
			}

			return sqlDB, nil
		}

		log.Warn("waiting for database", "attempt", attempt, "error", lastErr.Error())
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("database not ready after retries: %w", lastErr)
}

func waitForMySQLServer(dbCfg config.DatabaseConfig, log *slog.Logger) (*sql.DB, error) {
	var lastErr error
	for attempt := 1; attempt <= maxDBRetries; attempt++ {
		conn, err := openMySQLServer(dbCfg)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		log.Warn("waiting for database server", "attempt", attempt, "error", err.Error())
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("database server not ready after retries: %w", lastErr)
}

func buildPasswordCandidates(primary string) []string {
	candidates := []string{primary}
	add := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		for _, existing := range candidates {
			if existing == trimmed {
				return
			}
		}
		candidates = append(candidates, trimmed)
	}

	for _, item := range strings.Split(strings.TrimSpace(os.Getenv("DB_PASSWORD_FALLBACKS")), ",") {
		add(item)
	}

	return candidates
}

func openDatabaseWithPassword(base config.DatabaseConfig, password string) (*sql.DB, error) {
	cfg := base
	cfg.Password = password
	return db.NewMySQL(cfg)
}

func openMySQLServerWithPassword(base config.DatabaseConfig, password string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true&multiStatements=true",
		base.User,
		password,
		base.Host,
		base.Port,
	)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql admin connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping mysql admin connection: %w", err)
	}
	return conn, nil
}

func openMySQLServer(base config.DatabaseConfig) (*sql.DB, error) {
	passwordCandidates := buildPasswordCandidates(base.Password)
	var lastErr error
	for _, candidate := range passwordCandidates {
		conn, err := openMySQLServerWithPassword(base, candidate)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no password candidates available")
	}
	return nil, lastErr
}

func resetDatabase(ctx context.Context, dbCfg config.DatabaseConfig) error {
	dbName := strings.TrimSpace(dbCfg.Name)
	if dbName == "" {
		return fmt.Errorf("DB_NAME must be set for fresh reset")
	}

	adminDB, err := openMySQLServer(dbCfg)
	if err != nil {
		return fmt.Errorf("open mysql admin connection: %w", err)
	}
	defer adminDB.Close()

	quotedDBName := quoteIdentifier(dbName)
	resetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := adminDB.ExecContext(resetCtx, "DROP DATABASE IF EXISTS "+quotedDBName); err != nil {
		return fmt.Errorf("drop database %s: %w", dbName, err)
	}
	if _, err := adminDB.ExecContext(resetCtx, "CREATE DATABASE "+quotedDBName+" CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}

func ensureDatabaseExists(ctx context.Context, dbCfg config.DatabaseConfig) error {
	dbName := strings.TrimSpace(dbCfg.Name)
	if dbName == "" {
		return fmt.Errorf("DB_NAME must be set")
	}

	adminDB, err := openMySQLServer(dbCfg)
	if err != nil {
		return fmt.Errorf("open mysql admin connection: %w", err)
	}
	defer adminDB.Close()

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	createStmt := "CREATE DATABASE IF NOT EXISTS " + quoteIdentifier(dbName) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	if _, err := adminDB.ExecContext(queryCtx, createStmt); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}

	return nil
}

func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func syncRootPassword(ctx context.Context, sqlDB *sql.DB, newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return fmt.Errorf("configured root password is empty")
	}

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := sqlDB.QueryContext(queryCtx, "SELECT Host FROM mysql.user WHERE User = 'root'")
	if err != nil {
		return fmt.Errorf("query root hosts: %w", err)
	}
	defer rows.Close()

	hosts := make([]string, 0, 4)
	for rows.Next() {
		var host string
		if scanErr := rows.Scan(&host); scanErr != nil {
			return fmt.Errorf("scan root host: %w", scanErr)
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate root hosts: %w", err)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no root hosts found in mysql.user")
	}

	execCtx, execCancel := context.WithTimeout(ctx, 10*time.Second)
	defer execCancel()

	for _, host := range hosts {
		stmt := fmt.Sprintf(
			"ALTER USER '%s'@'%s' IDENTIFIED BY '%s'",
			escapeSQLString("root"),
			escapeSQLString(host),
			escapeSQLString(newPassword),
		)
		if _, err := sqlDB.ExecContext(execCtx, stmt); err != nil {
			return fmt.Errorf("alter root@%s password: %w", host, err)
		}
	}

	return nil
}

func escapeSQLString(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}

func runMigrations(cfg config.Config, source string) error {
	dsn := migrationDSN(cfg.Database)
	m, err := migrate.New(source, dsn)
	if err != nil {
		return fmt.Errorf("create migration instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}

		version, dirty, versionErr := m.Version()
		if versionErr == nil && dirty {
			if forceErr := m.Force(int(version)); forceErr != nil {
				return fmt.Errorf("force dirty migration version %d: %w", version, forceErr)
			}
			if retryErr := m.Up(); retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
				return fmt.Errorf("retry up migrations after dirty fix: %w", retryErr)
			}
			return nil
		}

		return fmt.Errorf("run up migrations: %w", err)
	}
	return nil
}

func resolveMigrationSource() (string, error) {
	if value := strings.TrimSpace(os.Getenv("MIGRATION_SOURCE")); value != "" {
		if strings.Contains(value, "://") {
			return value, nil
		}

		abs, err := filepath.Abs(value)
		if err != nil {
			return "", fmt.Errorf("resolve MIGRATION_SOURCE absolute path: %w", err)
		}
		return pathToFileSource(abs), nil
	}

	candidates := migrationDirCandidates()
	for _, candidate := range candidates {
		if dirExists(candidate) {
			return pathToFileSource(candidate), nil
		}
	}

	return "", fmt.Errorf("migrations directory not found; checked %d candidate paths", len(candidates))
}

func migrationDirCandidates() []string {
	candidates := make([]string, 0, 16)
	seen := map[string]struct{}{}

	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if _, exists := seen[abs]; exists {
			return
		}
		seen[abs] = struct{}{}
		candidates = append(candidates, abs)
	}

	cwd, err := os.Getwd()
	if err == nil {
		search := cwd
		for i := 0; i < 6; i++ {
			add(filepath.Join(search, "migrations"))
			add(filepath.Join(search, "backend", "migrations"))

			parent := filepath.Dir(search)
			if parent == search {
				break
			}
			search = parent
		}
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		add(filepath.Join(exeDir, "..", "migrations"))
		add(filepath.Join(exeDir, "..", "..", "migrations"))
		add(filepath.Join(exeDir, "..", "..", "backend", "migrations"))
	}

	return candidates
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func pathToFileSource(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func migrationDSN(dbCfg config.DatabaseConfig) string {
	password := url.QueryEscape(dbCfg.Password)
	return "mysql://" + dbCfg.User + ":" + password +
		"@tcp(" + dbCfg.Host + ":" + strconv.Itoa(dbCfg.Port) + ")/" + dbCfg.Name +
		"?multiStatements=true"
}

func envOrDefault(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}
