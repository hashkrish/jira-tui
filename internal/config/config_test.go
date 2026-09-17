package config

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "missing everything",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name: "valid basic auth config",
			cfg: Config{
				BaseURL:  "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token",
				AuthMode: "basic",
			},
			wantErr: false,
		},
		{
			name: "bearer mode does not require email",
			cfg: Config{
				BaseURL:  "https://jira.example.com",
				APIToken: "token",
				AuthMode: "bearer",
			},
			wantErr: false,
		},
		{
			name: "rejects non-http base url",
			cfg: Config{
				BaseURL:  "example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token",
				AuthMode: "basic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
