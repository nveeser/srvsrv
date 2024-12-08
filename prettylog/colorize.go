package prettylog

import (
	"fmt"
	"strconv"
)

const (
	reset = "\033[0m"

	black        textColor = 30
	red          textColor = 31
	green        textColor = 32
	yellow       textColor = 33
	blue         textColor = 34
	magenta      textColor = 35
	cyan         textColor = 36
	lightGray    textColor = 37
	darkGray     textColor = 90
	lightRed     textColor = 91
	lightGreen   textColor = 92
	lightYellow  textColor = 93
	lightBlue    textColor = 94
	lightMagenta textColor = 95
	lightCyan    textColor = 96
	white        textColor = 97
)

type textColor int

func colorize(colorCode textColor, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(int(colorCode)), v, reset)
}
