package db

import (
    "fmt"

    "github.com/rit3sh-x/blaze/core/constants"

    "github.com/jackc/pgx/v5/pgconn"
)

func (db *BlazeDB) Connect() bool {
    if db.Pool == nil {
        return false
    }

    var result int
    err := db.Pool.QueryRow(db.ctx, constants.TEST_QUERY).Scan(&result)
    return err == nil && result == 1
}

func (db *BlazeDB) Disconnect() bool {
    if db.Pool == nil {
        return false
    }

    db.Pool.Close()
    return true
}

func (db *BlazeDB) ExecuteRaw(query string, args ...interface{}) (int64, error) {
    if db.Pool == nil {
        return 0, fmt.Errorf("database connection is nil")
    }

    result, err := db.Pool.Exec(db.ctx, query, args...)
    if err != nil {
        return 0, err
    }

    return result.RowsAffected(), nil
}

func (db *BlazeDB) QueryRaw(query string, args ...interface{}) ([]map[string]interface{}, error) {
    if db.Pool == nil {
        return nil, fmt.Errorf("database connection is nil")
    }

    rows, err := db.Pool.Query(db.ctx, query, args...)
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
            row[fieldDescriptions[i].Name] = value
        }
        results = append(results, row)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return results, nil
}

func (db *BlazeDB) Transaction(queries []string, args [][]interface{}) ([]pgconn.CommandTag, error) {
    if db.Pool == nil {
        return nil, fmt.Errorf("database connection is nil")
    }

    if len(queries) != len(args) {
        return nil, fmt.Errorf("number of queries and arguments must match")
    }

    var results []pgconn.CommandTag

    tx, err := db.Pool.Begin(db.ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback(db.ctx)
            panic(p)
        } else if err != nil {
            tx.Rollback(db.ctx)
        } else {
            err = tx.Commit(db.ctx)
        }
    }()

    for i, query := range queries {
        result, execErr := tx.Exec(db.ctx, query, args[i]...)
        if execErr != nil {
            err = fmt.Errorf("failed to execute query %d: %w", i, execErr)
            return nil, err
        }
        results = append(results, result)
    }

    return results, nil
}