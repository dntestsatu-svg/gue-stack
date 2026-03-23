package config

import "testing"

func TestValidateJWTSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "placeholder access secret", value: "change-me-access-secret", wantErr: true},
		{name: "placeholder refresh secret", value: "change-me-refresh-secret", wantErr: true},
		{name: "too short", value: "short-secret-value", wantErr: true},
		{name: "strong secret", value: "this-is-a-very-strong-production-secret-12345", wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateJWTSecret("JWT_ACCESS_SECRET", tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
