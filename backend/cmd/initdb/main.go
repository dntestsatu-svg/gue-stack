package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
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
	migrationSource = "file:///app/migrations"
	maxDBRetries    = 30
	retryDelay      = 2 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	log := logger.New()
	sqlDB, err := waitForDatabase(cfg, log)
	if err != nil {
		log.Error("database is not ready", "error", err.Error())
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := runMigrations(cfg); err != nil {
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
	add("secret")

	return candidates
}

func openDatabaseWithPassword(base config.DatabaseConfig, password string) (*sql.DB, error) {
	cfg := base
	cfg.Password = password
	return db.NewMySQL(cfg)
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

func runMigrations(cfg config.Config) error {
	dsn := migrationDSN(cfg.Database)
	m, err := migrate.New(migrationSource, dsn)
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
