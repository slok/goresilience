package circuit

type contextKey string

func (c contextKey) String() string { return string(c) }
