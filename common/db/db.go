package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/config"
	"github.com/LexiconIndonesia/crawler-http-service/common/redis"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	zerolog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"

	"github.com/rs/zerolog/log"
)

// DB provides access to the database
type DB struct {
	Pool    *pgxpool.Pool
	Queries *repository.Queries
	Redis   *redis.RedisClient
}

// New creates a new DB instance
func New(pool *pgxpool.Pool, queries *repository.Queries, redis *redis.RedisClient) (*DB, error) {
	if pool == nil {
		return nil, errors.New("cannot use nil database pool")
	}
	if queries == nil {
		return nil, errors.New("cannot use nil queries")
	}
	return &DB{
		Pool:    pool,
		Queries: queries,
		Redis:   redis,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping checks if the database connection is alive
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// setupDatabase initializes the database connection
func SetupDatabase(ctx context.Context, cfg config.Config) (*DB, error) {
	config, err := pgxpool.ParseConfig(cfg.PgSql.ConnStr())
	if err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	// Configure connection pool for better performance and reliability
	config.MaxConns = 20                       // Maximum number of connections in the pool
	config.MinConns = 5                        // Minimum number of connections to maintain
	config.MaxConnLifetime = 30 * time.Minute  // Maximum lifetime of a connection
	config.MaxConnIdleTime = 5 * time.Minute   // Maximum idle time of a connection
	config.HealthCheckPeriod = 1 * time.Minute // How often to check connection health

	// Setup logger
	logger := zerolog.NewLogger(log.Logger)
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelInfo,
	}

	pgsqlClient, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Test connection
	if err := pgsqlClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	queries := repository.New(pgsqlClient)

	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating Redis client: %w", err)
	}

	// Create DB struct for dependency injection
	dbConn, err := New(pgsqlClient, queries, redisClient)
	if err != nil {
		return nil, fmt.Errorf("creating DB handler: %w", err)
	}

	return dbConn, nil
}
