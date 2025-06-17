package keycloak

import (
	"testing"

	"github.com/fulcrumproject/commons/auth"
)

func TestConfig_GetJWKSURL(t *testing.T) {
	config := &Config{
		KeycloakURL: "https://keycloak.example.com",
		Realm:       "test-realm",
	}

	expected := "https://keycloak.example.com/realms/test-realm/protocol/openid_connect/certs"
	actual := config.GetJWKSURL()

	if actual != expected {
		t.Errorf("Expected JWKS URL '%s', got '%s'", expected, actual)
	}
}

func TestConfig_GetIssuer(t *testing.T) {
	config := &Config{
		KeycloakURL: "https://keycloak.example.com",
		Realm:       "test-realm",
	}

	expected := "https://keycloak.example.com/realms/test-realm"
	actual := config.GetIssuer()

	if actual != expected {
		t.Errorf("Expected issuer '%s', got '%s'", expected, actual)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: &Config{
				KeycloakURL:  "https://keycloak.example.com",
				Realm:        "test-realm",
				ClientID:     "test-client",
				JWKSCacheTTL: 300,
			},
			expectError: false,
		},
		{
			name: "Empty KeycloakURL",
			config: &Config{
				Realm:        "test-realm",
				ClientID:     "test-client",
				JWKSCacheTTL: 300,
			},
			expectError: true,
			errorMsg:    "oauth keycloak URL cannot be empty when oauth authenticator is enabled",
		},
		{
			name: "Empty Realm",
			config: &Config{
				KeycloakURL:  "https://keycloak.example.com",
				ClientID:     "test-client",
				JWKSCacheTTL: 300,
			},
			expectError: true,
			errorMsg:    "oauth realm cannot be empty when oauth authenticator is enabled",
		},
		{
			name: "Empty ClientID",
			config: &Config{
				KeycloakURL:  "https://keycloak.example.com",
				Realm:        "test-realm",
				JWKSCacheTTL: 300,
			},
			expectError: true,
			errorMsg:    "oauth client ID cannot be empty when oauth authenticator is enabled",
		},
		{
			name: "Zero JWKSCacheTTL",
			config: &Config{
				KeycloakURL:  "https://keycloak.example.com",
				Realm:        "test-realm",
				ClientID:     "test-client",
				JWKSCacheTTL: 0,
			},
			expectError: true,
			errorMsg:    "oauth JWKS cache TTL must be positive when oauth authenticator is enabled",
		},
		{
			name: "Negative JWKSCacheTTL",
			config: &Config{
				KeycloakURL:  "https://keycloak.example.com",
				Realm:        "test-realm",
				ClientID:     "test-client",
				JWKSCacheTTL: -1,
			},
			expectError: true,
			errorMsg:    "oauth JWKS cache TTL must be positive when oauth authenticator is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAuthenticator_extractRole(t *testing.T) {
	config := &Config{
		ClientID: "test-client",
	}
	authenticator := &Authenticator{
		config: config,
	}

	tests := []struct {
		name         string
		claims       *Claims
		expectedRole auth.Role
		expectError  bool
	}{
		{
			name: "Direct role claim - admin",
			claims: &Claims{
				Role: "admin",
			},
			expectedRole: auth.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Direct role claim - participant",
			claims: &Claims{
				Role: "participant",
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Direct role claim - agent",
			claims: &Claims{
				Role: "agent",
			},
			expectedRole: auth.RoleAgent,
			expectError:  false,
		},
		{
			name: "Realm role - admin",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant", "admin", "user"},
				},
			},
			expectedRole: auth.RoleParticipant, // First valid role found
			expectError:  false,
		},
		{
			name: "Realm role - participant only",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Client role",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleAgent,
			expectError:  false,
		},
		{
			name: "Client role - multiple clients",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"other-client": {
						Roles: []string{"admin"},
					},
					"test-client": {
						Roles: []string{"participant"},
					},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "No valid role - invalid direct role",
			claims: &Claims{
				Role: "invalid-role",
			},
			expectError: true,
		},
		{
			name: "No valid role - invalid realm roles",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"invalid", "unknown"},
				},
			},
			expectError: true,
		},
		{
			name: "No valid role - invalid client roles",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"invalid"},
					},
				},
			},
			expectError: true,
		},
		{
			name:        "Empty claims",
			claims:      &Claims{},
			expectError: true,
		},
		{
			name: "Role priority - direct role takes precedence",
			claims: &Claims{
				Role: "admin",
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Role priority - realm role over client role",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Client role - wrong client ignored",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"wrong-client": {
						Roles: []string{"admin"},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := authenticator.extractRole(tt.claims)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if role != tt.expectedRole {
					t.Errorf("Expected role '%s', got '%s'", tt.expectedRole, role)
				}
			}
		})
	}
}
