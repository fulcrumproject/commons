package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

// Builder implements a generic builder pattern for creating configuration instances
type Builder[T any] struct {
	config    T
	err       error
	envPrefix string
	envTag    string
	envFiles  []string
}

// BuilderOption defines a function type for configuring the Builder
type BuilderOption[T any] func(*Builder[T])

// WithEnvPrefix sets the environment variable prefix
func WithEnvPrefix[T any](prefix string) BuilderOption[T] {
	return func(b *Builder[T]) {
		b.envPrefix = prefix
	}
}

// WithEnvTag sets the struct tag name for environment variables
func WithEnvTag[T any](tag string) BuilderOption[T] {
	return func(b *Builder[T]) {
		b.envTag = tag
	}
}

// WithEnvFiles sets the environment files to load
func WithEnvFiles[T any](files ...string) BuilderOption[T] {
	return func(b *Builder[T]) {
		b.envFiles = files
	}
}

// New returns a Builder with the provided default configuration and options
func New[T any](defaultConfig T, opts ...BuilderOption[T]) *Builder[T] {
	b := &Builder[T]{
		config:   defaultConfig,
		envTag:   "env", // Default tag
		envFiles: []string{},
	}

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// LoadFile loads configuration from a JSON file
func (b *Builder[T]) LoadFile(filepath *string) *Builder[T] {
	if b.err != nil {
		return b
	}

	if filepath == nil || *filepath == "" {
		return b
	}

	data, err := os.ReadFile(*filepath)
	if err != nil {
		b.err = fmt.Errorf("failed to read config file: %w", err)
		return b
	}

	if err := json.Unmarshal(data, b.config); err != nil {
		b.err = fmt.Errorf("failed to parse config file: %w", err)
		return b
	}

	return b
}

// WithEnv overrides configuration from environment variables using the stored configuration
func (b *Builder[T]) WithEnv() *Builder[T] {
	if b.err != nil {
		return b
	}

	err := loadEnvFromAncestors(b.envFiles...)
	if err != nil {
		b.err = fmt.Errorf("failed to load environment variables: %w", err)
		return b
	}

	if err := loadEnvToStruct(b.config, b.envPrefix, b.envTag); err != nil {
		b.err = fmt.Errorf("failed to override configuration from environment: %w", err)
		return b
	}

	return b
}

// Build validates and returns the final configuration
func (b *Builder[T]) Build() (T, error) {
	var zero T
	if b.err != nil {
		return zero, b.err
	}

	v := validator.New()
	if err := v.Struct(b.config); err != nil {
		return zero, fmt.Errorf("invalid configuration: %w", err)
	}

	return b.config, nil
}

// loadEnvToStruct loads environment variables into struct fields and nested structs based on tags
func loadEnvToStruct(target any, prefix, tag string) error {
	v := reflect.ValueOf(target).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get env tag or skip if not present
		// Check if field is a struct that needs recursive processing
		if fieldValue.Kind() == reflect.Struct {
			// Skip time.Duration which is technically a struct but should be treated as primitive
			if field.Type != reflect.TypeOf(time.Duration(0)) {
				if err := loadEnvToStruct(fieldValue.Addr().Interface(), prefix, tag); err != nil {
					return fmt.Errorf("error loading sub config field %s: %w", field.Name, err)
				}
			}
		}

		envVar, ok := field.Tag.Lookup(tag)
		if !ok || envVar == "" {
			continue
		}

		// Get value from environment or skip if empty
		envValue := os.Getenv(prefix + envVar)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				// Handle time.Duration
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(int64(duration))
			} else {
				// Handle regular integers
				val, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(val)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value for %s: %w", envVar, err)
			}
			fieldValue.SetUint(val)

		case reflect.Float32, reflect.Float64:
			val, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for %s: %w", envVar, err)
			}
			fieldValue.SetFloat(val)

		case reflect.Bool:
			val, err := strconv.ParseBool(envValue)
			if err != nil {
				return fmt.Errorf("invalid boolean value for %s: %w", envVar, err)
			}
			fieldValue.SetBool(val)

		case reflect.Slice:
			// Handle []string specifically. Add other slice types if needed.
			if fieldValue.Type().Elem().Kind() == reflect.String {
				parts := strings.Split(envValue, ",")
				// Trim spaces from each part
				for i, p := range parts {
					parts[i] = strings.TrimSpace(p)
				}
				fieldValue.Set(reflect.ValueOf(parts))
			}
		}
	}

	return nil
}

// loadEnvFromAncestors searches for .env files from the current directory up to the root
func loadEnvFromAncestors(filesToTry ...string) error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Track if we found any env files
	found := false

	// Start from current directory and move up
	dir := currentDir
	for {
		for _, fileName := range filesToTry {
			envPath := filepath.Join(dir, fileName)
			if _, err := os.Stat(envPath); err == nil {
				// File exists, load it
				if err := godotenv.Load(envPath); err == nil {
					slog.Info("Loading .env file", "file", envPath)
					found = true
				}
			}
		}

		// Stop if we've reached the root directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break // We've reached the root
		}
		dir = parentDir
	}

	if !found {
		slog.Info("No .env files found in ancestor directories")
	}

	return nil
}
