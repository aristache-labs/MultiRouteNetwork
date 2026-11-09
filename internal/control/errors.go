package control

import "errors"

var errNoEngine = errors.New("the routing engine is not running in this process")
