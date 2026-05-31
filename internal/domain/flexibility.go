package domain

type Flexibility int

const (
	Exact       Flexibility = 0
	Approximate Flexibility = 30
	Flexible    Flexibility = 60
)
