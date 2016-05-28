package server

import (
	_ "expvar"
	"github.com/Sirupsen/logrus"
	"net/http"
)

func Run(addr string) {
	logrus.Info("Started")
	http.ListenAndServe(addr, nil)
}
