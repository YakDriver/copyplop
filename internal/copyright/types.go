package copyright

type Issue struct {
	File    string
	Problem string
}

type FixResult struct {
	Fixed int
	Added int
}
