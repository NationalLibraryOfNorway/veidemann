// Copyright © 2017 National Library of Norway.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	stdcontext "context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/logger"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the persistent configuration.
type Config struct {
	Server             string        `yaml:"server,omitempty" mapstructure:"server"`
	RootCAs            string        `yaml:"certificate-authority-data,omitempty" mapstructure:"certificate-authority-data"`
	ServerNameOverride string        `yaml:"server-name-override,omitempty" mapstructure:"server-name-override"`
	AuthProvider       *AuthProvider `yaml:"auth-provider,omitempty" mapstructure:"auth-provider"`
}

// cfg is the persistent configuration.
var cfg Config

// AuthProvider is the authentication provider configuration.
type AuthProvider struct {
	// Name is the type of authentication provider.
	Name string `yaml:"name" mapstructure:"name"`

	// Config is the configuration for the authentication provider.
	Config any `yaml:"config" mapstructure:"config"`
}

const (
	ProviderOIDC   = "oidc"
	ProviderApiKey = "apikey"
)

// ApiKeyConfig is the configuration for the apikey authentication provider.
type ApiKeyConfig struct {
	ApiKey string `yaml:"api-key,omitempty" mapstructure:"api-key"`
}

// GetApiKeyConfig returns the apikey configuration.
func GetApiKeyConfig() (*ApiKeyConfig, error) {
	// If no auth provider in config, return nil
	if cfg.AuthProvider == nil {
		return nil, nil
	}
	// Use the api-key from the config file
	ap := new(ApiKeyConfig)
	// decode the config into the api-key config struct
	err := mapstructure.Decode(cfg.AuthProvider.Config, ap)
	return ap, err
}

// OIDCConfig is the configuration for the oidc authentication provider.
type OIDCConfig struct {
	ClientID     string   `yaml:"client-id" mapstructure:"client-id"`
	ClientSecret string   `yaml:"client-secret,omitempty" mapstructure:"client-secret"`
	Scopes       []string `yaml:"scopes,omitempty" mapstructure:"scopes"`
	IdToken      string   `yaml:"id-token" mapstructure:"id-token"`
	RefreshToken string   `yaml:"refresh-token,omitempty" mapstructure:"refresh-token"`
	IdpIssuerUrl string   `yaml:"idp-issuer-url" mapstructure:"idp-issuer-url"`
}

// GetOIDCConfig returns the oidc configuration.
func GetOIDCConfig() (*OIDCConfig, error) {
	if cfg.AuthProvider == nil {
		return nil, nil
	}
	ap := new(OIDCConfig)
	err := mapstructure.Decode(cfg.AuthProvider.Config, ap)
	return ap, err
}

// Init initializes configuration from config file, flags and environment variables.
func Init(flags *pflag.FlagSet) error {
	if flags == nil {
		return nil
	}

	logLevel, _ := flags.GetString("log-level")
	logFormat, _ := flags.GetString("log-format")
	logCaller, _ := flags.GetBool("log-caller")

	logger.InitLogger(logLevel, logFormat, logCaller)

	defer func() {
		slog.Debug("Using config file", "path", viper.ConfigFileUsed())
	}()

	// resolve path to context directory
	ctxDir, err := GetConfigPath("contexts")
	if err != nil {
		return fmt.Errorf("failed to resolve config directory path: %w", err)
	}

	// create context directory
	slog.Debug("Creating context directory", "path", ctxDir)
	if err := os.MkdirAll(ctxDir, 0777); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}
	// resolve context
	ctxName, _ := flags.GetString("context")
	ctxName, err = resolveContext(ctxName)
	if err != nil {
		return fmt.Errorf("failed to resolve context: %w", err)
	}
	// set runtime context
	viper.Set("context", ctxName)

	// resolve path to config file
	configFile, _ := flags.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(ctxDir)
		viper.SetConfigName(ctxName)
	}

	// read config file
	err = viper.ReadInConfig()
	if errors.As(err, new(viper.ConfigFileNotFoundError)) {
		path := filepath.Join(ctxDir, ctxName+".yaml")
		viper.SetConfigFile(path)
	} else if err != nil {
		return err
	}

	// store config in global variable
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return err
	}

	// bind flags to viper to
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
	if err := viper.BindPFlags(flags); err != nil {
		return fmt.Errorf("failed to bind flags: %w", err)
	}

	return nil
}

// GetContext returns the effective context.
func GetContext() string {
	return viper.GetString("context")
}

// GetServer returns the server address.
func GetServer() string {
	return viper.GetString("server")
}

// GetRootCAs returns certificate authority data.
func GetRootCAs() string {
	return viper.GetString("certificate-authority-data")
}

// GetServerNameOverride retuns
func GetServerNameOverride() string {
	return viper.GetString("server-name-override")
}

// GetAuthProviderName returns the authentication provider name.
func GetAuthProviderName() string {
	ap := GetAuthProvider()
	if ap == nil {
		return ""
	}
	return ap.Name
}

// GetAuthProvider returns the authentication provider.
func GetAuthProvider() *AuthProvider {
	return cfg.AuthProvider
}

func GetInsecure() bool {
	return viper.GetBool("insecure")
}

// SetServerAddress sets the server address.
func SetServerAddress(server string) error {
	cfg.Server = server
	return writeConfig()
}

// SetAuthProvider sets the authentication provider.
func SetApiKey(apiKey string) error {
	return SetAuthProvider(&AuthProvider{
		Name:   ProviderApiKey,
		Config: ApiKeyConfig{ApiKey: apiKey},
	})
}

// SetCaCert sets the certificate authority data.
func SetCaCert(cert string) error {
	cfg.RootCAs = cert
	return writeConfig()
}

// SetServerNameOverride sets the server name override.
func SetServerNameOverride(name string) error {
	cfg.ServerNameOverride = name
	return writeConfig()
}

// SetAuthProvider sets the authentication provider.
func SetAuthProvider(authProvider *AuthProvider) error {
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	return UpdateAuthProvider(ctx, func(*AuthProvider) (*AuthProvider, error) {
		return authProvider, nil
	})
}

// UpdateAuthProvider serializes auth provider changes across processes. The
// callback runs while the context config is locked and after it has been
// reloaded from disk.
func UpdateAuthProvider(ctx stdcontext.Context, update func(*AuthProvider) (*AuthProvider, error)) error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return errors.New("config file path is not initialized")
	}

	fileLock := flock.New(filepath.Join(filepath.Dir(configFile), "."+filepath.Base(configFile)+".lock"))
	locked, err := fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to lock config file %q: %w", configFile, err)
	}
	if !locked {
		return fmt.Errorf("timed out locking config file %q: %w", configFile, ctx.Err())
	}
	defer func() { _ = fileLock.Unlock() }()

	if _, err := os.Stat(configFile); err == nil {
		if err := reloadConfig(configFile); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect config file %q: %w", configFile, err)
	}

	authProvider, err := update(cfg.AuthProvider)
	if err != nil {
		return err
	}
	cfg.AuthProvider = authProvider
	return writeConfig()
}

func reloadConfig(configFile string) error {
	file, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	defer func() { _ = file.Close() }()
	var current Config
	if err := yaml.NewDecoder(file).Decode(&current); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to decode reloaded config: %w", err)
	}
	cfg = current
	return nil
}

// CreateContext creates a new context.
func CreateContext(name string) error {
	ok, err := ContextExists(name)
	if err != nil {
		return fmt.Errorf("failed to check if context '%s' already exists: %w", name, err)
	}
	if ok {
		return fmt.Errorf("context already exists: %s", name)
	}

	contextDir, err := GetConfigPath("contexts")
	if err != nil {
		return err
	}

	_, err = os.Create(filepath.Join(contextDir, name+".yaml"))
	return err
}

// writeConfig writes the config to file.
func writeConfig() error {
	y, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configFile := viper.ConfigFileUsed()

	dir := filepath.Dir(configFile)
	file, err := os.CreateTemp(dir, "."+filepath.Base(configFile)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file for \"%s\": %w", configFile, err)
	}
	tempName := file.Name()
	defer func() { _ = os.Remove(tempName) }()

	if err = file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to change access mode on config file \"%s\": %w", configFile, err)
	}

	if _, err = file.Write(y); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to write config file \"%s\": %w", configFile, err)
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to sync config file \"%s\": %w", configFile, err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("failed to close config file \"%s\": %w", configFile, err)
	}
	if err = os.Rename(tempName, configFile); err != nil {
		return fmt.Errorf("failed to replace config file \"%s\": %w", configFile, err)
	}
	return nil
}

// GetConfigPath returns the full path of a config directory or config file.
func GetConfigPath(subdirOrFile string) (string, error) {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".veidemann", subdirOrFile), nil
}

// context represents the context file.
type context struct {
	Context string
}

// resolveContext resolves the effective context.
func resolveContext(name string) (string, error) {
	switch name {
	case "":
		contextFile, err := GetConfigPath("context.yaml")
		if err != nil {
			return "", err
		}
		_, err = os.Stat(contextFile)
		if err != nil {
			err = SetCurrentContext("default")
			return "default", err
		}
		f, err := os.Open(contextFile)
		if err != nil {
			return "", err
		}
		defer func() { _ = f.Close() }()
		dec := yaml.NewDecoder(f)
		c := new(context)
		err = dec.Decode(c)
		if err != nil {
			return "", err
		}

		return c.Context, err
	case "kubectl":
		output, err := exec.Command("kubectl", "config", "current-context").CombinedOutput()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	default:
		return name, nil
	}
}

// SetCurrentContext sets the current context.
func SetCurrentContext(ctxName string) error {
	contextFile, err := GetConfigPath("context.yaml")
	if err != nil {
		return fmt.Errorf("failed to resolve config dir': %w", err)
	}
	w, err := os.Create(contextFile)
	if err != nil {
		return fmt.Errorf("failed to create or open '%s': %w", contextFile, err)
	}
	defer func() { _ = w.Close() }()

	enc := yaml.NewEncoder(w)
	err = enc.Encode(context{ctxName})
	if err != nil {
		return fmt.Errorf("failed to write context to '%s': %w", contextFile, err)
	}
	defer func() { _ = enc.Close() }()

	return nil
}

// ListContexts lists all contexts.
func ListContexts() ([]string, error) {
	var files []string
	contextDir, err := GetConfigPath("contexts")
	if err != nil {
		return nil, err
	}
	fileInfo, err := os.ReadDir(contextDir)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		if !file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			sufIdx := strings.LastIndex(file.Name(), ".")
			if sufIdx > 0 {
				files = append(files, file.Name()[:sufIdx])
			}
		}
	}
	return files, nil
}

// ContextExists checks if a context exists.
func ContextExists(name string) (bool, error) {
	cs, err := ListContexts()
	if err != nil {
		return false, err
	}

	for _, c := range cs {
		if name == c {
			return true, nil
		}
	}
	return false, nil
}

// GetConfig returns the persistent config.
func GetConfig() Config {
	return cfg
}
