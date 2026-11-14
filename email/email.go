package email

import (
	"encoding/json"
	"fmt"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"gopkg.in/yaml.v3"
)

type Email string

var emailRegex = regexp.MustCompile("^[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\\.)+[A-Za-z]{2,63}$")

func (e *Email) UnmarshalJSON(data []byte) error {
	var address string
	if err := json.Unmarshal(data, &address); err != nil {
		return fmt.Errorf("decode email: %w", err)
	}
	return e.UnmarshalText([]byte(address))
}

func (e *Email) UnmarshalText(data []byte) error {
	address := string(data)
	if len(address) > 254 || !emailRegex.MatchString(address) {
		return fmt.Errorf("invalid email address %q", address)
	}
	*e = Email(address)
	return nil
}

func (e *Email) UnmarshalBSONValue(valueType bsontype.Type, data []byte) error {
	address, ok := (bson.RawValue{Type: valueType, Value: data}).StringValueOK()
	if !ok {
		return fmt.Errorf("decode email: expected BSON string, got %s", valueType)
	}
	return e.UnmarshalText([]byte(address))
}

func (e *Email) UnmarshalYAML(node *yaml.Node) error {
	var address string
	if err := node.Decode(&address); err != nil {
		return fmt.Errorf("decode email: %w", err)
	}
	return e.UnmarshalText([]byte(address))
}

func (e *Email) Scan(value any) error {
	switch value := value.(type) {
	case string:
		return e.UnmarshalText([]byte(value))
	case []byte:
		return e.UnmarshalText(value)
	default:
		return fmt.Errorf("scan email: expected string or []byte, got %T", value)
	}
}
