package utils

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

func ReadFileClean(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if commentIndex := strings.Index(line, "//"); commentIndex != -1 {
			line = line[:commentIndex]
		}

		line = strings.TrimSpace(line)
		line = collapseSpaces(line)

		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

func ReadAndSeparateSchema(filePath string) (string, string, error) {
	content, err := ReadFileClean(filePath)
	if err != nil {
		return "", "", err
	}

	if strings.TrimSpace(content) == "" {
		return "", "", nil
	}

	enumContent, classContent := separateEnumsAndClassesWithRegex(content)
	return enumContent, classContent, nil
}

func separateEnumsAndClassesWithRegex(content string) (string, string) {
    enumPattern := regexp.MustCompile(`(?s)` + constants.KEYWORD_ENUM + `\s+[A-Z][a-zA-Z0-9_]{0,63}\s*\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)
    classPattern := regexp.MustCompile(`(?s)` + constants.KEYWORD_CLASS + `\s+[A-Z][a-zA-Z0-9_]{0,63}\s*\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)

    enumMatches := enumPattern.FindAllString(content, -1)
    classMatches := classPattern.FindAllString(content, -1)

    enumContent := strings.Join(enumMatches, "\n\n")
    classContent := strings.Join(classMatches, "\n\n")

    return enumContent, classContent
}

func collapseSpaces(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}