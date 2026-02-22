package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AskUser(question string) (string, error) {
	fmt.Print(fmt.Sprintf("%s: ", question))
	reader := bufio.NewReader(os.Stdin)

	text, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read from console: %w", err)
	}

	text = strings.Replace(text, "\n", "", -1)
	return text, nil
}
