package plugins

import ()

type SPI struct {
	DefaultPlugin
}

func init() {
	Register(&SPI{})
}
