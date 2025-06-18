package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	testUUID := properties.NewUUID()
	testIdentity := &auth.Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: auth.RoleAdmin,
	}

	tests := []struct {
		name               string
		authHeader         string
		authenticatorSetup func() *mockAuthenticator
		expectedStatus     int
		expectIdentity     bool
		expectedToken      string
	}{
		{
			name:       "Valid Bearer token",
			authHeader: "Bearer valid-token",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{
					identity: testIdentity,
					err:      nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectIdentity: true,
			expectedToken:  "valid-token",
		},
		{
			name:       "Missing Authorization header",
			authHeader: "",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{}
			},
			expectedStatus: http.StatusUnauthorized,
			expectIdentity: false,
			expectedToken:  "",
		},
		{
			name:       "Invalid token format - no Bearer prefix",
			authHeader: "invalid-token",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{}
			},
			expectedStatus: http.StatusUnauthorized,
			expectIdentity: false,
			expectedToken:  "",
		},
		{
			name:       "Authentication error",
			authHeader: "Bearer invalid-token",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{
					identity: nil,
					err:      errors.New("invalid token"),
				}
			},
			expectedStatus: http.StatusForbidden,
			expectIdentity: false,
			expectedToken:  "invalid-token",
		},
		{
			name:       "Nil identity returned",
			authHeader: "Bearer valid-token",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{
					identity: nil,
					err:      nil,
				}
			},
			expectedStatus: http.StatusForbidden,
			expectIdentity: false,
			expectedToken:  "valid-token",
		},
		{
			name:       "Bearer with empty token",
			authHeader: "Bearer ",
			authenticatorSetup: func() *mockAuthenticator {
				return &mockAuthenticator{
					identity: testIdentity,
					err:      nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectIdentity: true,
			expectedToken:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := tt.authenticatorSetup()

			// Create test handler that checks if identity is in context
			var capturedIdentity *auth.Identity
			var identityFound bool
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectIdentity {
					capturedIdentity = auth.MustGetIdentity(r.Context())
					identityFound = true
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := Auth(mockAuth)(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")

			if tt.expectIdentity {
				assert.True(t, identityFound, "Identity should be found in context")
				assert.Equal(t, testIdentity, capturedIdentity, "Identity should match expected")
				assert.True(t, mockAuth.called, "Authenticator should be called")
				assert.Equal(t, tt.expectedToken, mockAuth.receivedToken, "Token should be passed correctly")
			}
		})
	}
}

func TestAuthzFromExtractor(t *testing.T) {
	testUUID := properties.NewUUID()
	testIdentity := &auth.Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: auth.RoleAdmin,
	}

	tests := []struct {
		name            string
		extractorSetup  func() ObjectScopeExtractor
		authorizerSetup func() *mockAuthorizer
		expectedStatus  int
	}{
		{
			name: "Successful authorization",
			extractorSetup: func() ObjectScopeExtractor {
				return func(r *http.Request) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{
					err: nil,
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Extractor error",
			extractorSetup: func() ObjectScopeExtractor {
				return func(r *http.Request) (auth.ObjectScope, error) {
					return nil, errors.New("extraction failed")
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Authorization denied",
			extractorSetup: func() ObjectScopeExtractor {
				return func(r *http.Request) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{
					err: errors.New("access denied"),
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := tt.extractorSetup()
			mockAuthorizer := tt.authorizerSetup()

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := AuthzFromExtractor("user", "read", mockAuthorizer, extractor)(testHandler)

			// Create request with identity in context
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := auth.WithIdentity(req.Context(), testIdentity)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")
		})
	}
}

func TestIDScopeExtractor(t *testing.T) {
	testUUID := properties.NewUUID()
	testScope := &auth.AllwaysMatchObjectScope{}

	tests := []struct {
		name        string
		loaderSetup func() ObjectScopeLoader
		expectError bool
		errorMsg    string
	}{
		{
			name: "Successful scope loading",
			loaderSetup: func() ObjectScopeLoader {
				return func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return testScope, nil
				}
			},
			expectError: false,
		},
		{
			name: "Loader error",
			loaderSetup: func() ObjectScopeLoader {
				return func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return nil, errors.New("resource not found")
				}
			},
			expectError: true,
			errorMsg:    "cannot load resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := tt.loaderSetup()
			extractor := IDScopeExtractor(loader)

			// Create request with UUID in context
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := context.WithValue(req.Context(), uuidContextKey, testUUID)
			req = req.WithContext(ctx)

			scope, err := extractor(req)

			if tt.expectError {
				require.Error(t, err, "Expected an error")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				assert.Nil(t, scope, "Scope should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, testScope, scope, "Scope should match expected")
			}
		})
	}
}

func TestAuthzFromID(t *testing.T) {
	testUUID := properties.NewUUID()
	testIdentity := &auth.Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: auth.RoleAdmin,
	}

	tests := []struct {
		name            string
		loaderSetup     func() ObjectScopeLoader
		authorizerSetup func() *mockAuthorizer
		expectedStatus  int
	}{
		{
			name: "Successful authorization with ID",
			loaderSetup: func() ObjectScopeLoader {
				return func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{err: nil}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Loader fails",
			loaderSetup: func() ObjectScopeLoader {
				return func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return nil, errors.New("resource not found")
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := tt.loaderSetup()
			mockAuthorizer := tt.authorizerSetup()

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := AuthzFromID("user", "read", mockAuthorizer, loader)(testHandler)

			// Create request with identity and UUID in context
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := auth.WithIdentity(req.Context(), testIdentity)
			ctx = context.WithValue(ctx, uuidContextKey, testUUID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")
		})
	}
}

func TestSimpleScopeExtractor(t *testing.T) {
	extractor := SimpleScopeExtractor()

	req := httptest.NewRequest("GET", "/test", nil)
	scope, err := extractor(req)

	assert.NoError(t, err, "Should not return an error")
	assert.NotNil(t, scope, "Scope should not be nil")
	assert.IsType(t, &auth.AllwaysMatchObjectScope{}, scope, "Should return AllwaysMatchObjectScope")
}

func TestAuthzSimple(t *testing.T) {
	testUUID := properties.NewUUID()
	testIdentity := &auth.Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: auth.RoleAdmin,
	}

	tests := []struct {
		name            string
		authorizerSetup func() *mockAuthorizer
		expectedStatus  int
	}{
		{
			name: "Successful simple authorization",
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{err: nil}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Authorization denied",
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{err: errors.New("access denied")}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthorizer := tt.authorizerSetup()

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := AuthzSimple("user", "read", mockAuthorizer)(testHandler)

			// Create request with identity in context
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := auth.WithIdentity(req.Context(), testIdentity)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")
		})
	}
}

func TestBodyScopeExtractor(t *testing.T) {
	testScope := &auth.AllwaysMatchObjectScope{}

	tests := []struct {
		name        string
		bodySetup   func() *mockObjectScopeProvider
		expectError bool
		errorMsg    string
	}{
		{
			name: "Successful scope extraction from body",
			bodySetup: func() *mockObjectScopeProvider {
				return &mockObjectScopeProvider{
					scope: testScope,
					err:   nil,
				}
			},
			expectError: false,
		},
		{
			name: "Body scope extraction error",
			bodySetup: func() *mockObjectScopeProvider {
				return &mockObjectScopeProvider{
					scope: nil,
					err:   errors.New("invalid scope"),
				}
			},
			expectError: true,
			errorMsg:    "invalid auth scope in request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.bodySetup()
			extractor := BodyScopeExtractor[*mockObjectScopeProvider]()

			// Create request with body in context
			req := httptest.NewRequest("POST", "/test", nil)
			ctx := context.WithValue(req.Context(), decodedBodyContextKey, body)
			req = req.WithContext(ctx)

			scope, err := extractor(req)

			if tt.expectError {
				require.Error(t, err, "Expected an error")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				assert.Nil(t, scope, "Scope should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, testScope, scope, "Scope should match expected")
			}
		})
	}
}

func TestAuthzFromBody(t *testing.T) {
	testUUID := properties.NewUUID()
	testIdentity := &auth.Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: auth.RoleAdmin,
	}

	tests := []struct {
		name            string
		bodySetup       func() *mockObjectScopeProvider
		authorizerSetup func() *mockAuthorizer
		expectedStatus  int
	}{
		{
			name: "Successful authorization from body",
			bodySetup: func() *mockObjectScopeProvider {
				return &mockObjectScopeProvider{
					scope: &auth.AllwaysMatchObjectScope{},
					err:   nil,
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{err: nil}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Body scope extraction fails",
			bodySetup: func() *mockObjectScopeProvider {
				return &mockObjectScopeProvider{
					scope: nil,
					err:   errors.New("invalid scope"),
				}
			},
			authorizerSetup: func() *mockAuthorizer {
				return &mockAuthorizer{}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.bodySetup()
			mockAuthorizer := tt.authorizerSetup()

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware
			middleware := AuthzFromBody[*mockObjectScopeProvider]("user", "write", mockAuthorizer)(testHandler)

			// Create request with identity and body in context
			req := httptest.NewRequest("POST", "/test", nil)
			ctx := auth.WithIdentity(req.Context(), testIdentity)
			ctx = context.WithValue(ctx, decodedBodyContextKey, body)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Execute middleware
			middleware.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected")
		})
	}
}

// Mock implementations for testing

type mockAuthenticator struct {
	identity      *auth.Identity
	err           error
	called        bool
	receivedCtx   context.Context
	receivedToken string
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, token string) (*auth.Identity, error) {
	m.called = true
	m.receivedCtx = ctx
	m.receivedToken = token
	return m.identity, m.err
}

type mockAuthorizer struct {
	err error
}

func (m *mockAuthorizer) Authorize(identity *auth.Identity, action auth.Action, object auth.ObjectType, objectScope auth.ObjectScope) error {
	return m.err
}

type mockObjectScopeProvider struct {
	scope auth.ObjectScope
	err   error
}

func (m *mockObjectScopeProvider) ObjectScope() (auth.ObjectScope, error) {
	return m.scope, m.err
}
