package types

type Node struct {
	Id   uint32
	Addr string
}

type ChangeListener interface {
	Changed([]Node)
}
