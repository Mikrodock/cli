package utils

import (
	"bufio"
	"strings"
)

func DetectOSType(osRelease string) OSType {
	scanner := bufio.NewScanner(strings.NewReader(osRelease))
	for scanner.Scan() {
		line := scanner.Text()
		splitted := strings.Split(line, "=")
		if len(splitted) == 2 && splitted[0] == "ID" {
			switch {
			case strings.Contains(splitted[1], "ubuntu"):
				return Ubuntu
			case strings.Contains(splitted[1], "debian"):
				return Debian
			case strings.Contains(splitted[1], "arch"):
				return ArchLinux
			case strings.Contains(splitted[1], "boot2docker"):
				return Boot2Docker
			case strings.Contains(splitted[1], "alpine"):
				return Alpine
			default:
				return Unknown
			}
		}
	}
	return Unknown
}
