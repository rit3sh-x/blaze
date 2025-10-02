package initblaze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

func Init() error {
	if _, err := os.Stat(constants.PROJECT_DIR); err == nil {
		return fmt.Errorf(constants.RED+"project directory %q already exists"+constants.RESET, constants.PROJECT_DIR)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf(constants.RED+"failed to check directory: %w"+constants.RESET, err)
	}

	dirs := []string{
		constants.PROJECT_DIR,
		constants.MIGRATION_DIR,
		constants.CLIENT_DIR,
		constants.TYPES_DIR,
		constants.PROVIDER_DIR,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf(constants.RED+"failed to create directory %q: %w"+constants.RESET, dir, err)
		}
	}

	files := []struct {
		path    string
		content []byte
	}{
		{constants.SCHEMA_FILE, []byte("")},
	}

	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.path), 0755); err != nil {
			return fmt.Errorf(constants.RED+"failed to create parent dir for %q: %w"+constants.RESET, file.path, err)
		}

		if err := os.WriteFile(file.path, file.content, 0644); err != nil {
			return fmt.Errorf(constants.RED+"failed to write file %q: %w"+constants.RESET, file.path, err)
		}
	}

	gitignorePath := ".gitignore"
	gitignoreContent := []byte(fmt.Sprintf("\n# Blaze project\n/%s\n", constants.CLIENT_DIR))
	entry := fmt.Sprintf("/%s\n", constants.CLIENT_DIR)

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, gitignoreContent, 0644); err != nil {
			return fmt.Errorf(constants.RED+"failed to create .gitignore file: %w"+constants.RESET, err)
		}
	} else {
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf(constants.RED+"failed to read .gitignore file: %w"+constants.RESET, err)
		}
		if !strings.Contains(string(data), entry) {
			file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf(constants.RED+"failed to open .gitignore file: %w"+constants.RESET, err)
			}
			defer file.Close()

			if _, err := file.WriteString(string(gitignoreContent)); err != nil {
				return fmt.Errorf(constants.RED+"failed to append to .gitignore file: %w"+constants.RESET, err)
			}
		}
	}

	fmt.Printf(constants.GREEN+"✔ Blaze project initialized at ./%s"+constants.RESET+"\n", constants.PROJECT_DIR)
	return nil
}