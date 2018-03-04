package utils

type OSType string

const (
	Ubuntu      OSType = "Ubuntu"
	Debian      OSType = "Debian"
	ArchLinux   OSType = "Arch Linux"
	Boot2Docker OSType = "Boot2Docker"
	Alpine      OSType = "Alpine Linux"
	Unknown     OSType = "Unknown"
)
