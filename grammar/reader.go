package grammar

import (
	"bufio"
	"os"
	"strings"

	"go.uber.org/zap"
)

func ReadInputGrammarFile(filePath string, log *zap.SugaredLogger) []string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Error(err)
		return nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err = scanner.Err(); err != nil {
		log.Error(err)
		return nil
	}

	return lines
}
