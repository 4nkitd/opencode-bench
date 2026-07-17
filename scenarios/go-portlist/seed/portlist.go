package portlist

type Listener struct {
	PID     int
	Command string
	Port    int
	Addr    string
}

func ParseLsof(out string) []Listener {
	panic("not implemented")
}
