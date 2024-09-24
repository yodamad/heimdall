package commons

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func AskQuestion(question string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(question)
	rawInput, _ := reader.ReadString(EOL_BYTE)
	return strings.TrimRight(rawInput, EOL)
}
