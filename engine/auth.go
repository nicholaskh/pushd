package engine

import (
	"errors"
	"fmt"
	cmap "github.com/nicholaskh/golib/concurrent/map"
	log "github.com/nicholaskh/log4go"
)

var (
	userToken  cmap.ConcurrentMap = cmap.New() //username => token
	loginUsers cmap.ConcurrentMap = cmap.New() //username => 1
)

//TODO maybe sasl is more secure
func authStep(name, token string) (string, error) {
	if token, exists := userToken.Get(name); exists && token == token {
		userToken.Remove(name)
		loginUsers.Set(name, 1)
		return fmt.Sprintf("Auth succeed"), nil
	} else {
		return "", errors.New("Auth fail")
	}

}

func checkLogin(name string) error {
	log.Debug("%s %s", name, loginUsers)
	if loginUsers.Has(name) {
		return nil
	} else {
		return errors.New(fmt.Sprintf("%s not login", name))
	}
}
