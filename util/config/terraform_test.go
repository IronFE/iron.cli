package config

import (
	"reflect"
	"testing"
)

func TestTerraformConfig_Merge(t *testing.T) {
	tests := []struct {
		name     string
		base     TerraformConfig
		other    TerraformConfig
		expected TerraformConfig
	}{
		{
			name: "Merge TerraformVersion",
			base: TerraformConfig{
				TerraformVersion: "1.0.0",
			},
			other: TerraformConfig{
				TerraformVersion: "1.1.0",
			},
			expected: TerraformConfig{
				TerraformVersion: "1.1.0",
			},
		},
		{
			name: "Merge Backend",
			base: TerraformConfig{
				Backend: Backend{
					Type: "s3",
					Config: map[string]string{
						"bucket": "old-bucket",
					},
				},
			},
			other: TerraformConfig{
				Backend: Backend{
					Type: "gcs",
					Config: map[string]string{
						"bucket": "new-bucket",
					},
				},
			},
			expected: TerraformConfig{
				Backend: Backend{
					Type: "gcs",
					Config: map[string]string{
						"bucket": "new-bucket",
					},
				},
			},
		},
		{
			name: "Merge Providers - New Provider",
			base: TerraformConfig{
				Providers: []*Provider{
					{Name: "aws"},
				},
			},
			other: TerraformConfig{
				Providers: []*Provider{
					{Name: "google"},
				},
			},
			expected: TerraformConfig{
				Providers: []*Provider{
					{Name: "aws"},
					{Name: "google"},
				},
			},
		},
		{
			name: "Merge Providers - Existing Provider Update",
			base: TerraformConfig{
				Providers: []*Provider{
					{
						Name:   "aws",
						Source: "hashicorp/aws",
						Config: map[string]string{"region": "us-east-1"},
						Tags:   map[string]string{"env": "dev"},
					},
				},
			},
			other: TerraformConfig{
				Providers: []*Provider{
					{
						Name:   "aws",
						Source: "custom/aws",
						Config: map[string]string{"profile": "default"},
						Tags:   map[string]string{"owner": "me"},
					},
				},
			},
			expected: TerraformConfig{
				Providers: []*Provider{
					{
						Name:   "aws",
						Source: "custom/aws",
						Config: map[string]string{"region": "us-east-1", "profile": "default"},
						Tags:   map[string]string{"env": "dev", "owner": "me"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.other)
			if !reflect.DeepEqual(tt.base, tt.expected) {
				t.Errorf("Merge() = %v, want %v", tt.base, tt.expected)
			}
		})
	}
}
