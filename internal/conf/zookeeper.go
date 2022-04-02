package conf

import (
	"github.com/api7/apisix-seed/internal/utils"
	"gopkg.in/yaml.v3"
)

func init() {
	DisBuilders["zookeeper"] = zkBuilder
}

const zkConfSchema = `
{
  "type": "object",
  "properties": {
    "Hosts": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "string",
        "pattern": "^[a-zA-Z0-9-_.:\\@]+$",
        "minLength": 2,
        "maxLength": 100
      }
    },
    "Prefix": {
      "type": "string",
      "pattern": "^[\\/a-zA-Z0-9-_.]*$",
      "maxLength": 100
    },
    "Weight": {
      "type": "integer",
      "minimum": 1,
      "default": 100
    },
    "Timeout": {
      "type": "integer",
      "minimum": 1,
      "default": 100
    }
  },
  "required": [
    "Hosts"
  ]
}
`

type Zookeeper struct {
	Hosts   []string
	Prefix  string
	Weight  int
	Timeout int
}

func zkBuilder(content []byte) (interface{}, error) {
	zookeeper := Zookeeper{
		Weight:  100,
		Timeout: 10,
	}
	err := yaml.Unmarshal(content, &zookeeper)
	if err != nil {
		return nil, err
	}

	validator, err := utils.NewJsonSchemaValidator(zkConfSchema)
	if err != nil {
		return nil, err
	}

	if err = validator.Validate(zookeeper); err != nil {
		return nil, err
	}
	return &zookeeper, nil
}
