package drivers

type State int

const (
	NotCreated State = iota
	InCreation
	Running
	Stopped
	Stuck
	Unknown
)
