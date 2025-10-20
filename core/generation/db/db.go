package db

import (
	"fmt"
	"strings"
)

func GenerateDBUtils() string {
	var content strings.Builder

	content.WriteString(`package client

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/rit3sh-x/blaze/core/constants"
    "github.com/rit3sh-x/blaze/core/db"
)

type BlazeDatabaseClient struct {
    *db.BlazeDB
    ctx context.Context
}

func DB(ctx context.Context, envFile string) (*BlazeDatabaseClient, error) {
    blazeDB, err := db.DB(ctx, envFile)
    if err != nil {
        return nil, err
    }
    return &BlazeDatabaseClient{
        BlazeDB: blazeDB,
        ctx: ctx,
    }, nil
}

func (c *BlazeDatabaseClient) WithContext(ctx context.Context) *BlazeDatabaseClient {
    return &BlazeDatabaseClient{
        BlazeDB: c.BlazeDB,
        ctx: ctx,
    }
}

func (c *BlazeDatabaseClient) Context() context.Context {
    if c.ctx == nil {
        return context.Background()
    }
    return c.ctx
}

func (c *BlazeDatabaseClient) Connect() bool {
    if c.Pool == nil {
        return false
    }

    var result int
    err := c.Pool.QueryRow(c.Context(), constants.TEST_QUERY).Scan(&result)
    return err == nil && result == 1
}

func (c *BlazeDatabaseClient) Disconnect() bool {
    if c.Pool == nil {
        return false
    }

    c.Pool.Close()
    return true
}

func (c *BlazeDatabaseClient) ExecuteRaw(query string, args ...interface{}) (int64, error) {
    if c.Pool == nil {
        return 0, fmt.Errorf("database connection is nil")
    }

    result, err := c.Pool.Exec(c.Context(), query, args...)
    if err != nil {
        return 0, err
    }

    return result.RowsAffected(), nil
}

func (c *BlazeDatabaseClient) QueryRaw(query string, args ...interface{}) ([]map[string]interface{}, error) {
    if c.Pool == nil {
        return nil, fmt.Errorf("database connection is nil")
    }

    rows, err := c.Pool.Query(c.Context(), query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []map[string]interface{}
    fieldDescriptions := rows.FieldDescriptions()

    for rows.Next() {
        values, err := rows.Values()
        if err != nil {
            return nil, err
        }

        row := make(map[string]interface{})
        for i, value := range values {
            row[string(fieldDescriptions[i].Name)] = value
        }
        results = append(results, row)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return results, nil
}

func (c *BlazeDatabaseClient) Transaction(queries []string, args [][]interface{}) ([]pgconn.CommandTag, error) {
    if c.Pool == nil {
        return nil, fmt.Errorf("database connection is nil")
    }

    if len(queries) != len(args) {
        return nil, fmt.Errorf("number of queries and arguments must match")
    }

    var results []pgconn.CommandTag

    tx, err := c.Pool.Begin(c.Context())
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback(c.Context())
            panic(p)
        } else if err != nil {
            tx.Rollback(c.Context())
        } else {
            err = tx.Commit(c.Context())
        }
    }()

    for i, query := range queries {
        result, execErr := tx.Exec(c.Context(), query, args[i]...)
        if execErr != nil {
            err = fmt.Errorf("failed to execute query %d: %w", i, execErr)
            return nil, err
        }
        results = append(results, result)
    }

    return results, nil
}`)

	return content.String()
}

func GenerateClientAccessors(classNames []string) string {
	var content strings.Builder

	content.WriteString("\n// ==================== MODEL CLIENT ACCESSORS ====================\n\n")

	for _, className := range classNames {
		content.WriteString(fmt.Sprintf("func (c *BlazeDatabaseClient) %s() *%sClient {\n", className, className))
		content.WriteString(fmt.Sprintf("\treturn &%sClient{db: c}\n", className))
		content.WriteString("}\n\n")
	}

	return content.String()
}