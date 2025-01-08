package util

type Empty struct{}

type NewModel interface {
	NewModel() interface{}
}
