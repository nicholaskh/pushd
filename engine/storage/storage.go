package storage

type msgTuple struct {
	channel string
	msg     string
}

var storage struct {
	driver *storageDriver
}

import (
	"labix.org/v2/mgo"
)

var (
	msgQueue chan msgTuple
)

func store() {

}
