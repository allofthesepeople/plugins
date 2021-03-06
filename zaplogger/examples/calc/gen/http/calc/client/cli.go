// Code generated by goa v2.0.0-wip, DO NOT EDIT.
//
// calc HTTP client CLI support package
//
// Command:
// $ goa gen goa.design/plugins/zaplogger/examples/calc/design -o
// $(GOPATH)/src/goa.design/plugins/zaplogger/examples/calc

package client

import (
	"fmt"
	"strconv"

	calcsvc "goa.design/plugins/zaplogger/examples/calc/gen/calc"
)

// BuildAddPayload builds the payload for the calc add endpoint from CLI flags.
func BuildAddPayload(calcAddA string, calcAddB string) (*calcsvc.AddPayload, error) {
	var err error
	var a int
	{
		var v int64
		v, err = strconv.ParseInt(calcAddA, 10, 64)
		a = int(v)
		if err != nil {
			err = fmt.Errorf("invalid value for a, must be INT")
		}
	}
	var b int
	{
		var v int64
		v, err = strconv.ParseInt(calcAddB, 10, 64)
		b = int(v)
		if err != nil {
			err = fmt.Errorf("invalid value for b, must be INT")
		}
	}
	if err != nil {
		return nil, err
	}
	payload := &calcsvc.AddPayload{
		A: a,
		B: b,
	}
	return payload, nil
}
