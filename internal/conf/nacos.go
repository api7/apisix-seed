package conf

import (
	"github.com/api7/apisix-seed/internal/utils"
	"gopkg.in/yaml.v3"
	"log"
)

func init() {
	DisBuilders["nacos"] = nacosBuilder
}

const schema = `
{
  "type": "object",
  "properties": {
    "Host": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "string",
        "pattern": "^http(s)?:\\/\\/[a-zA-Z0-9-_.:]+$",
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
    "User": {
      "type": "string",
      "pattern": "^[a-zA-Z0-9-_.]*$",
      "maxLength": 100
    },
    "Password": {
      "type": "string",
      "pattern": "^[a-zA-Z0-9-_.]*$",
      "maxLength": 100
    },
    "Timeout": {
      "type": "object",
      "properties": {
        "Connect": {
          "type": "integer",
          "minimum": 1,
          "default": 2000
        },
        "Send": {
          "type": "integer",
          "minimum": 1,
          "default": 2000
        },
        "Read": {
          "type": "integer",
          "minimum": 1,
          "default": 5000
        }
      }
    }
  },
  "required": [
    "Host"
  ]
}
`

type timeout struct {
	Connect int
	Send    int
	Read    int
}

type Nacos struct {
	Host     []string
	Prefix   string
	User     string
	Password string
	Weight   int
	Timeout  timeout
}

func nacosBuilder(content []byte) (interface{}, error) {
	// go jsonschema lib doesn't support setting default values
	// so we need to set for some default fields ourselves.
	nacos := Nacos{
		Weight: 100,
		Timeout: timeout{
			Connect: 2000,
			Send:    2000,
			Read:    5000,
		},
	}
	err := yaml.Unmarshal(content, &nacos)
	if err != nil {
		return nil, err
	}

	validator, err := utils.NewJsonSchemaValidator(schema)
	if err != nil {
		return nil, err
	}

	if err = validator.Validate(nacos); err != nil {
		log.Printf("================= levy err: %s", err.Error())
		return nil, err
	}
	return &nacos, nil
}
