package storage

type msgTuple struct {
	channel string
	msg     string
}

type storageDriver interface {
	func store()
}

var storage struct {
	driver *storageDriver
}

var (
	msgQueue chan msgTuple
)

func store() {

}
