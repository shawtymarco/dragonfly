package cmd

// Operator reports whether src is a server operator. Console sources that
// implement Operator() bool returning true are treated as operators. Players
// expose this through player.Player.Operator.
func Operator(src Source) bool {
	type operatorSource interface {
		Operator() bool
	}
	o, ok := src.(operatorSource)
	return ok && o.Operator()
}
