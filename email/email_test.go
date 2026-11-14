package email

import (
	"database/sql"
	"encoding"
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v3"
)

func TestEmailUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Email
		valid bool
	}{
		{"simple", `"user@example.com"`, "user@example.com", true},
		{"subdomain and plus tag", `"first.last+tag@mail.example.co.uk"`, "first.last+tag@mail.example.co.uk", true},
		{"missing at sign", `"user.example.com"`, "", false},
		{"missing domain", `"user@"`, "", false},
		{"double dot", `"first..last@example.com"`, "", false},
		{"invalid domain label", `"user@-example.com"`, "", false},
		{"surrounding whitespace", `" user@example.com "`, "", false},
		{"empty", `""`, "", false},
		{"null", `null`, "", false},
		{"non-string", `42`, "", false},
		{"malformed JSON", `"user@example.com`, "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address := Email("original@example.com")
			err := json.Unmarshal([]byte(test.input), &address)
			if test.valid {
				if err != nil || address != test.want {
					t.Fatalf("address = %q, error = %v; want %q", address, err, test.want)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if address != "original@example.com" {
				t.Fatalf("address changed after error: %q", address)
			}
		})
	}
}

func TestEmailUnmarshalText(t *testing.T) {
	var _ encoding.TextUnmarshaler = (*Email)(nil)

	address := Email("original@example.com")
	if err := address.UnmarshalText([]byte("new@example.com")); err != nil || address != "new@example.com" {
		t.Fatalf("address = %q, error = %v", address, err)
	}
	if err := address.UnmarshalText([]byte("invalid")); err == nil || address != "new@example.com" {
		t.Fatalf("invalid text: address = %q, error = %v", address, err)
	}
}

func TestEmailScan(t *testing.T) {
	var _ sql.Scanner = (*Email)(nil)

	for _, value := range []any{"new@example.com", []byte("new@example.com")} {
		address := Email("original@example.com")
		if err := address.Scan(value); err != nil || address != "new@example.com" {
			t.Fatalf("Scan(%T): address = %q, error = %v", value, address, err)
		}
	}
	for _, value := range []any{"invalid", []byte("invalid"), nil, 42} {
		address := Email("original@example.com")
		if err := address.Scan(value); err == nil || address != "original@example.com" {
			t.Fatalf("Scan(%T): address = %q, error = %v", value, address, err)
		}
	}
}

func TestEmailBSON(t *testing.T) {
	var _ bson.ValueUnmarshaler = (*Email)(nil)

	for _, test := range []struct {
		name  string
		value any
		valid bool
	}{
		{"valid string", "new@example.com", true},
		{"invalid string", "invalid", false},
		{"number", 42, false},
		{"null", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := bson.Marshal(bson.M{"address": test.value})
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Address Email `bson:"address"`
			}
			result.Address = "original@example.com"
			err = bson.Unmarshal(data, &result)
			if test.valid {
				if err != nil || result.Address != "new@example.com" {
					t.Fatalf("address = %q, error = %v", result.Address, err)
				}
			} else if err == nil || result.Address != "original@example.com" {
				t.Fatalf("address = %q, error = %v", result.Address, err)
			}
		})
	}
}

func TestEmailYAML(t *testing.T) {
	var _ yaml.Unmarshaler = (*Email)(nil)

	for _, test := range []struct {
		name      string
		input     string
		valid     bool
		skipField bool
	}{
		{"valid string", "address: new@example.com\n", true, false},
		{"invalid string", "address: invalid\n", false, false},
		{"number", "address: 42\n", false, false},
		{"null", "address: null\n", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var result struct {
				Address Email `yaml:"address"`
			}
			result.Address = "original@example.com"
			err := yaml.Unmarshal([]byte(test.input), &result)
			if test.skipField {
				if err != nil || result.Address != "original@example.com" {
					t.Fatalf("address = %q, error = %v", result.Address, err)
				}
				return
			}
			if test.valid {
				if err != nil || result.Address != "new@example.com" {
					t.Fatalf("address = %q, error = %v", result.Address, err)
				}
			} else if err == nil || result.Address != "original@example.com" {
				t.Fatalf("address = %q, error = %v", result.Address, err)
			}
		})
	}
}
