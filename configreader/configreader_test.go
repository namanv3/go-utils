package configreader

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadYamlConfig(t *testing.T) {
	unsetEnv(t, "CONNECTION_URL", "CONNECTION_PORT", "MONGO_PASSWORD", "STARTING_MESSAGE", "optionalField")
	t.Setenv("MONGO_URL", "mongodb://localhost:27017")

	type config struct {
		Connection struct {
			URL  string `yaml:"url"`
			Port int    `yaml:"port"`
		} `yaml:"connection"`
		Database struct {
			Type string `yaml:"type"`
			Spec struct {
				URL      string `yaml:"url"`
				Username string `yaml:"username"`
				Password string `yaml:"password"`
			} `yaml:"spec"`
		} `yaml:"database"`
		StartingMessage string `yaml:"startingMessage"`
		Optionally      string `yaml:"optionally"`
	}

	path := filepath.Join("testdata", "test.yaml")
	t.Run("defaults and relative path", func(t *testing.T) {
		got, err := ReadYamlConfig[config](path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Connection.URL != "0.0.0.0" || got.Connection.Port != 8080 {
			t.Errorf("connection = %+v", got.Connection)
		}
		if got.Database.Type != "mongodb" || got.Database.Spec.URL != "mongodb://localhost:27017" ||
			got.Database.Spec.Username != "admin" || got.Database.Spec.Password != "admin" {
			t.Errorf("database = %+v", got.Database)
		}
		if got.StartingMessage != "this is a message for you" || got.Optionally != "" {
			t.Errorf("strings = %q, %q", got.StartingMessage, got.Optionally)
		}
	})

	t.Run("environment overrides and absolute path", func(t *testing.T) {
		// A set, empty environment variable takes precedence over a default.
		t.Setenv("MONGO_PASSWORD", "")
		t.Setenv("CONNECTION_PORT", "9090")
		absolute, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ConfigReader[config](absolute)
		if err != nil {
			t.Fatal(err)
		}
		if got.Database.Spec.Password != "" {
			t.Errorf("password = %q, want empty", got.Database.Spec.Password)
		}
		if got.Connection.Port != 9090 {
			t.Errorf("port = %d, want 9090", got.Connection.Port)
		}
	})
}

func TestReadJsonConfig(t *testing.T) {
	t.Setenv("JSON_HOST", "host\"with\\quotes\nand newline")
	t.Setenv("JSON_PORT", "9000")
	unsetEnv(t, "JSON_GREETING", "JSON_EMPTY")

	path := filepath.Join("testdata", "test.json")
	type config struct {
		Server struct {
			Host string `json:"host"`
			Port int    `json:"port,string"`
		} `json:"server"`
		Messages []string `json:"messages"`
	}
	t.Run("defaults and relative path", func(t *testing.T) {
		got, err := ReadJsonConfig[config](path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Server.Host != "host\"with\\quotes\nand newline" || got.Server.Port != 9000 {
			t.Errorf("server = %+v", got.Server)
		}
		wantMessages := []string{"before host\"with\\quotes\nand newline after", "hello there", ""}
		if !reflect.DeepEqual(got.Messages, wantMessages) {
			t.Errorf("messages = %#v, want %#v", got.Messages, wantMessages)
		}
	})

	t.Run("environment overrides and absolute path", func(t *testing.T) {
		t.Setenv("JSON_GREETING", "updated greeting")
		t.Setenv("JSON_EMPTY", "filled")
		absolute, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ConfigReader[config](absolute)
		if err != nil {
			t.Fatal(err)
		}
		wantMessages := []string{"before host\"with\\quotes\nand newline after", "updated greeting", "filled"}
		if !reflect.DeepEqual(got.Messages, wantMessages) {
			t.Errorf("messages = %#v, want %#v", got.Messages, wantMessages)
		}
	})
}

func TestNumericValuesRemainStrings(t *testing.T) {
	t.Setenv("NUMERIC_STRING_ENV", "67890")
	unsetEnv(t, "NUMERIC_STRING_DEFAULT")

	type config struct {
		FromEnv     string `yaml:"fromEnv" json:"fromEnv"`
		FromDefault string `yaml:"fromDefault" json:"fromDefault"`
	}
	for _, file := range []string{"numeric_strings.yaml", "numeric_strings.json"} {
		t.Run(file, func(t *testing.T) {
			got, err := ConfigReader[config](filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			if got.FromEnv != "67890" || got.FromDefault != "12345" {
				t.Errorf("config = %+v, want numeric text in both string fields", got)
			}
		})
	}
}

func TestConfigReaderReportsAllMissingVariables(t *testing.T) {
	unsetEnv(t, "MISSING_ALPHA", "MISSING_BETA")
	tests := []struct {
		name string
		path string
	}{
		{"YAML", filepath.Join("testdata", "missing.yaml")},
		{"JSON", filepath.Join("testdata", "missing.json")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConfigReader[map[string]any](test.path)
			if err == nil || !strings.Contains(err.Error(), "MISSING_ALPHA, MISSING_BETA") {
				t.Fatalf("error = %v, want both missing variables once in sorted order", err)
			}
		})
	}
}

func TestConfigReaderErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"unsupported extension", filepath.Join("testdata", "errors", "unsupported.txt"), "unsupported"},
		{"invalid YAML", filepath.Join("testdata", "errors", "invalid.yaml"), "parse YAML"},
		{"invalid JSON", filepath.Join("testdata", "errors", "invalid.json"), "parse JSON"},
		{"extra JSON value", filepath.Join("testdata", "errors", "extra.json"), "multiple JSON values"},
		{"YAML type mismatch", filepath.Join("testdata", "errors", "wrong_type.yaml"), "decode YAML"},
		{"JSON type mismatch", filepath.Join("testdata", "errors", "wrong_type.json"), "decode JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConfigReader[struct {
				Port int `yaml:"port" json:"port"`
			}](test.path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	_, err := ConfigReader[struct{}](filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("error = %v, want read error", err)
	}
}

func unsetEnv(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		value, present := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				os.Setenv(name, value)
			} else {
				os.Unsetenv(name)
			}
		})
	}
}
