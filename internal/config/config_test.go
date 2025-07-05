package config

import (
	"os"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		envs map[string]string
		args []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "Error_ServerHost_Required",
			args: args{
				args: []string{"-a="},
			},
			wantErr: true,
		},
		{
			name: "Check_Flags_Priority",
			args: args{
				envs: map[string]string{
					"RUN_ADDRESS":            ":18090",
					"LOG_LEVEL":              "error",
					"DATABASE_URI":           "postgres://postgres:postgres@localhost:15429/loyalty?sslmode=disable",
					"SECRET":                 "secret_secret",
					"ACCRUAL_SYSTEM_ADDRESS": "https://test-accrual/",
				},
				args: []string{
					"-a=:18080",
					"-l=error",
					"-d=postgres://postgres:postgres@227.0.0.1:15429/loyalty?sslmode=disable",
					"-s=secret_secret_priority",
					"-r=https://test-accrual-priority/",
				},
			},
			want: &Config{
				ServerHost:        ":18080",
				LogLevel:          "error",
				DatabaseDSN:       "postgres://postgres:postgres@227.0.0.1:15429/loyalty?sslmode=disable",
				AuthSecret:        "secret_secret_priority",
				AccrualHost:       "https://test-accrual-priority/",
				ActualizeInterval: defaultActualizeInterval,
			},
			wantErr: false,
		},
		{
			name: "Check_Envs_Set",
			args: args{
				envs: map[string]string{
					"RUN_ADDRESS":            ":18090",
					"LOG_LEVEL":              "error",
					"DATABASE_URI":           "postgres://postgres:postgres@localhost:15429/loyalty?sslmode=disable",
					"SECRET":                 "secret_secret",
					"ACCRUAL_SYSTEM_ADDRESS": "https://test-accrual/",
				},
				args: []string{},
			},
			want: &Config{
				ServerHost:        ":18090",
				LogLevel:          "error",
				DatabaseDSN:       "postgres://postgres:postgres@localhost:15429/loyalty?sslmode=disable",
				AuthSecret:        "secret_secret",
				AccrualHost:       "https://test-accrual/",
				ActualizeInterval: defaultActualizeInterval,
			},
			wantErr: false,
		},
		{
			name: "Check_Flags_Set",
			args: args{
				envs: map[string]string{},
				args: []string{
					"-a=:28090",
					"-l=fatal",
					"-d=postgres://postgres:postgres@127.0.0.1:15429/loyalty?sslmode=disable",
					"-s=secret_secret_flags",
					"-r=https://test-accrual-flags/",
				},
			},
			want: &Config{
				ServerHost:        ":28090",
				LogLevel:          "fatal",
				DatabaseDSN:       "postgres://postgres:postgres@127.0.0.1:15429/loyalty?sslmode=disable",
				AuthSecret:        "secret_secret_flags",
				AccrualHost:       "https://test-accrual-flags/",
				ActualizeInterval: defaultActualizeInterval,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envs != nil {
				for k, v := range tt.args.envs {
					if err := os.Setenv(k, v); err != nil {
						t.Fatal(err)
					}
				}
			}

			got, err := New(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() got = %v, want %v", got, tt.want)
			}
		})
	}
}
