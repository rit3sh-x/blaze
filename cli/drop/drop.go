package drop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/jackc/pgx/v5/pgxpool"
)

func DropProject(pool *pgxpool.Pool) error {
	if err := dropFiles(); err != nil {
		return err
	}

	if err := dropDatabase(pool); err != nil {
		return err
	}

	fmt.Printf(constants.GREEN + "✔ Blaze project fully reset (files + database)\n" + constants.RESET)
	return nil
}

func dropFiles() error {
	if _, err := os.Stat(constants.PROJECT_DIR); os.IsNotExist(err) {
		return fmt.Errorf(constants.RED+"project directory %q does not exist"+constants.RESET, constants.PROJECT_DIR)
	}

	dirs := []string{
		constants.MIGRATION_DIR,
		constants.CLIENT_DIR,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf(constants.RED+"failed to read directory %q: %w"+constants.RESET, dir, err)
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf(constants.RED+"failed to remove %q: %w"+constants.RESET, path, err)
			}
		}

		if len(entries) > 0 {
			fmt.Printf(constants.GREEN+"Cleared contents of %s"+constants.RESET+"\n", dir)
		}
	}

	if _, err := os.Stat(constants.SCHEMA_FILE); err == nil {
		if err := os.WriteFile(constants.SCHEMA_FILE, []byte(""), 0644); err != nil {
			return fmt.Errorf(constants.RED+"failed to clear file %q: %w"+constants.RESET, constants.SCHEMA_FILE, err)
		}
		fmt.Printf(constants.GREEN+"Emptied file %s"+constants.RESET+"\n", constants.SCHEMA_FILE)
	}

	return nil
}

func dropDatabase(pool *pgxpool.Pool) error {
	ctx := context.Background()

	_, err := pool.Exec(ctx, constants.DROP_PUBLIC_SCHEMA)
	if err != nil {
		return fmt.Errorf(constants.RED+"failed to reset public schema: %w"+constants.RESET, err)
	}

	fmt.Printf(constants.GREEN + "Cleared all tables, types, indexes, and enums in public schema\n" + constants.RESET)
	return nil
}