package util

import (
	"fmt"
)

type Conv[ConvArg any, To any] interface {
	To(ConvArg) (To, error)
}

type AbstactConv interface {
	AbstactTo(any) (any, error)
}

type AbstactConvWrapper[ConvArg any, To any] struct {
	from Conv[ConvArg, To]
}

func (a AbstactConvWrapper[ConvArg, To]) AbstactTo(arg any) (any, error) {
	// Try to assert that the argument is of type ConvArg.
	var convargModel ConvArg
	convArg, ok := arg.(ConvArg)
	if !ok {
		return nil, fmt.Errorf("argument type %T does not match expected type %T", arg, convargModel)
	}

	// Perform the conversion.
	result, err := a.from.To(convArg)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ToAbstactConv[ConvArg any, To any](c Conv[ConvArg, To]) AbstactConv {
	return AbstactConvWrapper[ConvArg, To]{from: c}
}
