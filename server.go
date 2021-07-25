package gcat

type Server interface {
	Serve() error
}
