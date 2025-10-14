package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/rit3sh-x/blaze/core/constants"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type BlazeDB struct {
	Pool *pgxpool.Pool
	ctx  context.Context
}

func DB(ctx context.Context, envFile string, varName string) (*BlazeDB, error) {
	minConns := 0
	maxConns := 25

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("%sNo env file found at %s, using OS environment variables%s\n", constants.YELLOW, envFile, constants.RESET)
	} else {
		fmt.Printf("%sLoaded environment variables from %s%s\n", constants.GREEN, envFile, constants.RESET)
	}

	dbURI := os.Getenv(varName)
	if dbURI == "" {
		return nil, fmt.Errorf("%sEnvironment variable %q not set%s", constants.RED, varName, constants.RESET)
	}

	config, err := pgxpool.ParseConfig(dbURI)
	if err != nil {
		return nil, fmt.Errorf("%sFailed to parse database URI: %v%s", constants.RED, err, constants.RESET)
	}

	config.MinConns = int32(minConns)
	config.MaxConns = int32(maxConns)

	if maxConns := os.Getenv(constants.DB_MAX_CONNS_ENV); maxConns != "" {
		if val, err := strconv.Atoi(maxConns); err == nil {
			config.MaxConns = int32(val)
		}
	}
	if minConns := os.Getenv(constants.DB_MIN_CONNS_ENV); minConns != "" {
		if val, err := strconv.Atoi(minConns); err == nil {
			config.MinConns = int32(val)
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("%sFailed to create connection pool: %v%s", constants.RED, err, constants.RESET)
	}

	var ping int
	if err := pool.QueryRow(ctx, constants.TEST_QUERY).Scan(&ping); err != nil {
		return nil, fmt.Errorf("%sFailed to ping database: %v%s", constants.RED, err, constants.RESET)
	}

	fmt.Printf("%s✔ Connected to database%s\n", constants.GREEN, constants.RESET)
	return &BlazeDB{Pool: pool, ctx: ctx}, nil
}

func (db *BlazeDB) WithContext(ctx context.Context) *BlazeDB {
	return &BlazeDB{
		Pool: db.Pool,
		ctx:  ctx,
	}
}

func (db *BlazeDB) Context() context.Context {
	return db.ctx
}