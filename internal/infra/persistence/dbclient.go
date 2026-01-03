package persistence

type Client interface {
	DB() DB
	Close() error
}
