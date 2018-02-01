package vagrant

import (
	"bufio"
	"strings"
)

func GetValue(message string, key string) []string {
	scanner := bufio.NewScanner(strings.NewReader(message))
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), ",")
		msgType := data[2]
		args := data[3:]
		if key == msgType {
			return args
		}
	}

	return nil

}
