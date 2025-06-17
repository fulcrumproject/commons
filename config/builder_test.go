package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig is a simple test configuration for testing the generic builder
type testConfig struct {
	Name     string   `json:"name" env:"NAME"`
	Port     int      `json:"port" env:"PORT"`
	Enabled  bool     `json:"enabled" env:"ENABLED"`
	Factor   float64  `json:"factor" env:"FACTOR"`
	MaxConns uint     `json:"max_conns" env:"MAX_CONNS"`
	Tags     []string `json:"tags" env:"TAGS"`
}

func newTestConfig() *testConfig {
	return &testConfig{
		Name:     "test-app",
		Port:     8080,
		Enabled:  true,
		Factor:   1.0,
		MaxConns: 100,
		Tags:     []string{"default"},
	}
}

func (c *testConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if c.Factor < 0 {
		return fmt.Errorf("factor cannot be negative")
	}
	return nil
}

// Helper function to set and clean up environment variables
func setEnvVars(t *testing.T, envVars map[string]string) {
	t.Helper()
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	t.Cleanup(func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	})
}

func TestGenericBuilder_Default(t *testing.T) {
	cfg, err := New(newTestConfig()).Build()
	require.NoError(t, err)

	assert.Equal(t, "test-app", cfg.Name)
	assert.Equal(t, 8080, cfg.Port)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 1.0, cfg.Factor)
	assert.Equal(t, []string{"default"}, cfg.Tags)
}

func TestGenericBuilder_LoadFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid JSON",
			content: `{
				"name": "file-app",
				"port": 9090,
				"enabled": false,
				"factor": 2.5,
				"tags": ["file", "tag"]
			}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			content:     `{invalid json}`,
			expectError: true,
			errorMsg:    "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			cfg, err := New(newTestConfig()).LoadFile(&configPath).Build()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "file-app", cfg.Name)
				assert.Equal(t, 9090, cfg.Port)
				assert.False(t, cfg.Enabled)
				assert.Equal(t, 2.5, cfg.Factor)
				assert.Equal(t, []string{"file", "tag"}, cfg.Tags)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := "/path/to/nonexistent/config.json"
		_, err := New(newTestConfig()).LoadFile(&nonExistentFile).Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	// Test nil and empty filepath
	t.Run("nil and empty filepath", func(t *testing.T) {
		cfg, err := New(newTestConfig()).LoadFile(nil).Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name)

		emptyPath := ""
		cfg, err = New(newTestConfig()).LoadFile(&emptyPath).Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name)
	})
}

func TestGenericBuilder_WithEnv(t *testing.T) {
	setEnvVars(t, map[string]string{
		"TEST_NAME":    "env-app",
		"TEST_PORT":    "7070",
		"TEST_ENABLED": "false",
		"TEST_FACTOR":  "3.14",
		"TEST_TAGS":    "env,tag1,tag2",
	})

	cfg, err := New(newTestConfig(), WithEnvPrefix[*testConfig]("TEST_"), WithEnvTag[*testConfig]("env")).WithEnv().Build()
	require.NoError(t, err)

	assert.Equal(t, "env-app", cfg.Name)
	assert.Equal(t, 7070, cfg.Port)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 3.14, cfg.Factor)
	assert.Equal(t, []string{"env", "tag1", "tag2"}, cfg.Tags)
}

func TestGenericBuilder_ChainedOperations(t *testing.T) {
	// Create a temporary file
	configJSON := `{"name": "file-app", "port": 9090, "factor": 2.0}`
	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	setEnvVars(t, map[string]string{
		"TEST_PORT": "6060",
		"TEST_TAGS": "env,tag",
	})

	// Chain operations: default -> file -> env
	cfg, err := New(newTestConfig(), WithEnvPrefix[*testConfig]("TEST_"), WithEnvTag[*testConfig]("env")).
		LoadFile(&configPath).
		WithEnv().
		Build()
	require.NoError(t, err)

	assert.Equal(t, "file-app", cfg.Name)             // From file
	assert.Equal(t, 6060, cfg.Port)                   // From env
	assert.True(t, cfg.Enabled)                       // From default
	assert.Equal(t, 2.0, cfg.Factor)                  // From file
	assert.Equal(t, []string{"env", "tag"}, cfg.Tags) // From env
}

func TestGenericBuilder_ErrorPropagation(t *testing.T) {
	// Test that errors are properly propagated through the chain
	nonExistentFile := "/nonexistent/file.json"
	builder := New(newTestConfig()).LoadFile(&nonExistentFile)

	// Further operations should not execute when there's an error
	result := builder.WithEnv()
	assert.Equal(t, builder, result) // Should return same builder

	_, err := builder.Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

// Test types for comprehensive coverage
type NestedConfig struct {
	Value string `env:"VALUE"`
}

type TestConfigWithNested struct {
	Name   string `env:"NAME"`
	Nested NestedConfig
}

func (c *TestConfigWithNested) Validate() error { return nil }

type TestConfigWithUnexported struct {
	Name            string `env:"NAME"`
	unexportedField string `env:"UNEXPORTED"` // This should be skipped
}

func (c *TestConfigWithUnexported) Validate() error { return nil }

type TestConfigNoTags struct {
	Name     string `env:"NAME"`
	NoTag    string // No env tag
	EmptyTag string `env:""`
}

func (c *TestConfigNoTags) Validate() error { return nil }

type TestConfigAllTypes struct {
	Name       string        `env:"NAME"`
	Port       int           `env:"PORT"`
	UintVal    uint          `env:"UINT_VAL"`
	Float32Val float32       `env:"FLOAT32_VAL"`
	Duration   time.Duration `env:"DURATION"`
	Enabled    bool          `env:"ENABLED"`
	Tags       []string      `env:"TAGS"`
}

func (c *TestConfigAllTypes) Validate() error { return nil }

func TestGenericBuilder_StructProcessing(t *testing.T) {
	t.Run("nested structs", func(t *testing.T) {
		setEnvVars(t, map[string]string{"TEST_VALUE": "nested-value"})

		defaultConfig := &TestConfigWithNested{
			Name:   "test",
			Nested: NestedConfig{Value: "default"},
		}

		cfg, err := New(defaultConfig, WithEnvPrefix[*TestConfigWithNested]("TEST_"), WithEnvTag[*TestConfigWithNested]("env")).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "nested-value", cfg.Nested.Value)
	})

	t.Run("unexported fields", func(t *testing.T) {
		setEnvVars(t, map[string]string{
			"TEST_NAME":       "new-name",
			"TEST_UNEXPORTED": "should-be-ignored",
		})

		defaultConfig := &TestConfigWithUnexported{
			Name:            "test",
			unexportedField: "default",
		}

		cfg, err := New(defaultConfig, WithEnvPrefix[*TestConfigWithUnexported]("TEST_"), WithEnvTag[*TestConfigWithUnexported]("env")).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "new-name", cfg.Name)
		assert.Equal(t, "default", cfg.unexportedField) // Should remain unchanged
	})

	t.Run("fields without env tags", func(t *testing.T) {
		setEnvVars(t, map[string]string{"TEST_NAME": "new-name"})

		defaultConfig := &TestConfigNoTags{
			Name:     "test",
			NoTag:    "default-no-tag",
			EmptyTag: "default-empty-tag",
		}

		cfg, err := New(defaultConfig, WithEnvPrefix[*TestConfigNoTags]("TEST_"), WithEnvTag[*TestConfigNoTags]("env")).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "new-name", cfg.Name)
		assert.Equal(t, "default-no-tag", cfg.NoTag)       // Should remain unchanged
		assert.Equal(t, "default-empty-tag", cfg.EmptyTag) // Should remain unchanged
	})
}

func TestGenericBuilder_DataTypes(t *testing.T) {
	setEnvVars(t, map[string]string{
		"TEST_NAME":        "env-name",
		"TEST_PORT":        "9090",
		"TEST_UINT_VAL":    "200",
		"TEST_FLOAT32_VAL": "6.28",
		"TEST_DURATION":    "5m30s",
		"TEST_ENABLED":     "false",
		"TEST_TAGS":        "env,tag1,tag2",
	})

	defaultConfig := &TestConfigAllTypes{
		Name:       "test",
		Port:       8080,
		UintVal:    100,
		Float32Val: 3.14,
		Duration:   time.Second * 30,
		Enabled:    true,
		Tags:       []string{"default"},
	}

	cfg, err := New(defaultConfig, WithEnvPrefix[*TestConfigAllTypes]("TEST_"), WithEnvTag[*TestConfigAllTypes]("env")).WithEnv().Build()
	require.NoError(t, err)

	assert.Equal(t, "env-name", cfg.Name)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, uint(200), cfg.UintVal)
	assert.Equal(t, float32(6.28), cfg.Float32Val)
	assert.Equal(t, time.Minute*5+time.Second*30, cfg.Duration)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, []string{"env", "tag1", "tag2"}, cfg.Tags)
}

func TestGenericBuilder_EnvParsingErrors(t *testing.T) {
	tests := []struct {
		name        string
		envVarKey   string
		envVarValue string
		expectedErr string
	}{
		{
			name:        "invalid int",
			envVarKey:   "TEST_PORT",
			envVarValue: "not-an-int",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid bool",
			envVarKey:   "TEST_ENABLED",
			envVarValue: "not-a-bool",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid float",
			envVarKey:   "TEST_FACTOR",
			envVarValue: "not-a-float",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid uint",
			envVarKey:   "TEST_MAX_CONNS",
			envVarValue: "-1",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid duration",
			envVarKey:   "TEST_DURATION",
			envVarValue: "invalid-duration",
			expectedErr: "invalid duration value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t, map[string]string{tt.envVarKey: tt.envVarValue})

			var defaultConfig interface{ Validate() error }
			if tt.envVarKey == "TEST_DURATION" {
				defaultConfig = &TestConfigAllTypes{}
			} else {
				defaultConfig = newTestConfig()
			}

			switch cfg := defaultConfig.(type) {
			case *testConfig:
				_, err := New(cfg, WithEnvPrefix[*testConfig]("TEST_"), WithEnvTag[*testConfig]("env")).WithEnv().Build()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			case *TestConfigAllTypes:
				_, err := New(cfg, WithEnvPrefix[*TestConfigAllTypes]("TEST_"), WithEnvTag[*TestConfigAllTypes]("env")).WithEnv().Build()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestGenericBuilder_EmptyEnvValues(t *testing.T) {
	setEnvVars(t, map[string]string{
		"TEST_NAME":   "",
		"TEST_FACTOR": "",
		"TEST_PORT":   "7070",
	})

	defaultConfig := newTestConfig()
	defaultConfig.Tags = []string{"default", "tag"}

	cfg, err := New(defaultConfig, WithEnvPrefix[*testConfig]("TEST_"), WithEnvTag[*testConfig]("env")).WithEnv().Build()
	require.NoError(t, err)

	// Should keep default values for empty env vars
	assert.Equal(t, "test-app", cfg.Name)
	assert.Equal(t, 1.0, cfg.Factor)
	// Port should be updated
	assert.Equal(t, 7070, cfg.Port)
	// Tags should remain default as it's not set in env
	assert.Equal(t, []string{"default", "tag"}, cfg.Tags)
}

func TestGenericBuilder_WithEnvFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create env files
	envFile1Content := "TEST_NAME=name-from-file1\nTEST_PORT=1111"
	envFile1Path := filepath.Join(tempDir, ".env.test1")
	err := os.WriteFile(envFile1Path, []byte(envFile1Content), 0644)
	require.NoError(t, err)

	envFile2Content := "TEST_PORT=2222\nTEST_FACTOR=7.7"
	envFile2Path := filepath.Join(tempDir, ".env.test2")
	err = os.WriteFile(envFile2Path, []byte(envFile2Content), 0644)
	require.NoError(t, err)

	// Set actual environment variables
	setEnvVars(t, map[string]string{
		"TEST_NAME": "name-from-actual-env",
		"TEST_TAGS": "actual,env",
	})

	// Change working directory to tempDir
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	t.Cleanup(func() { os.Chdir(originalWD) })

	cfg, err := New(newTestConfig(),
		WithEnvPrefix[*testConfig]("TEST_"),
		WithEnvTag[*testConfig]("env"),
		WithEnvFiles[*testConfig](".env.test1", ".env.test2"),
	).WithEnv().Build()
	require.NoError(t, err)

	assert.Equal(t, "name-from-actual-env", cfg.Name)    // From actual env
	assert.Equal(t, 1111, cfg.Port)                      // From .env.test1 (first file wins)
	assert.True(t, cfg.Enabled)                          // Default value
	assert.Equal(t, 7.7, cfg.Factor)                     // From .env.test2
	assert.Equal(t, []string{"actual", "env"}, cfg.Tags) // From actual env
}

func TestGenericBuilder_SliceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single value",
			envValue: "single",
			expected: []string{"single"},
		},
		{
			name:     "values with spaces",
			envValue: " tag1 , tag2 , tag3 ",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty values in list",
			envValue: "tag1,,tag3",
			expected: []string{"tag1", "", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t, map[string]string{"TEST_TAGS": tt.envValue})

			defaultConfig := &TestConfigAllTypes{Tags: []string{"default"}}
			cfg, err := New(defaultConfig, WithEnvPrefix[*TestConfigAllTypes]("TEST_"), WithEnvTag[*TestConfigAllTypes]("env")).WithEnv().Build()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Tags)
		})
	}
}

func TestGenericBuilder_EdgeCases(t *testing.T) {
	t.Run("error propagation with existing error", func(t *testing.T) {
		// Create a builder with an existing error
		nonExistentFile := "/nonexistent/file.json"
		builder := New(newTestConfig()).LoadFile(&nonExistentFile)

		// Try to load another file - should be skipped due to existing error
		validJSON := `{"name": "should-not-load"}`
		validConfigPath := filepath.Join(t.TempDir(), "valid_config.json")
		err := os.WriteFile(validConfigPath, []byte(validJSON), 0644)
		require.NoError(t, err)

		result := builder.LoadFile(&validConfigPath)
		assert.Equal(t, builder, result) // Should return same builder

		_, err = result.Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file") // Original error preserved
	})

	t.Run("env files edge cases", func(t *testing.T) {
		// Test with empty filesToTry slice
		cfg, err := New(newTestConfig(), WithEnvFiles[*testConfig]()).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name)

		// Test with non-existent files (should not cause errors)
		cfg, err = New(newTestConfig(), WithEnvFiles[*testConfig]("non-existent-1.env", "non-existent-2.env")).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name)
	})

	t.Run("godotenv load error handling", func(t *testing.T) {
		tempDir := t.TempDir()
		originalWD, err := os.Getwd()
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)
		t.Cleanup(func() { os.Chdir(originalWD) })

		// Create a directory instead of a file to cause godotenv.Load to fail gracefully
		envDirPath := filepath.Join(tempDir, ".env")
		err = os.Mkdir(envDirPath, 0755)
		require.NoError(t, err)

		// This should still work because godotenv.Load error is handled gracefully
		cfg, err := New(newTestConfig(), WithEnvFiles[*testConfig](".env")).WithEnv().Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name) // Should use default values
	})
}
