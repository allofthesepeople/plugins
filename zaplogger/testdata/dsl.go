package testdata

import (
	. "goa.design/goa/dsl"
)

var SimpleServiceDSL = func() {
	Service("SimpleService", func() {
		Method("SimpleMethod", func() {
			HTTP(func() {
				GET("/")
			})
		})
	})
}
