package utils

import (
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap/buffer"
)

type Validator interface {
	Validate(obj interface{}) error
}

type JsonSchemaValidator struct {
	schema *gojsonschema.Schema
}

func NewJsonSchemaValidator(bs string) (Validator, error) {
	s, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(bs))
	if err != nil {
		return nil, fmt.Errorf("new schema failed: %s", err)
	}
	return &JsonSchemaValidator{
		schema: s,
	}, nil
}

func (v *JsonSchemaValidator) Validate(obj interface{}) error {
	ret, err := v.schema.Validate(gojsonschema.NewGoLoader(obj))
	if err != nil {
		return fmt.Errorf("validate failed: %s", err)
	}

	if !ret.Valid() {
		errString := buffer.Buffer{}
		for i, vErr := range ret.Errors() {
			if i != 0 {
				errString.AppendString("\n")
			}
			errString.AppendString(vErr.String())
		}
		return errors.New(errString.String())
	}
	return nil
}
