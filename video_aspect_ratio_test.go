package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_getVideoAspectRatio(t *testing.T) {
	r, _ := os.Getwd()
	dir := filepath.Join(r, "samples")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "Horizontal",
			path:    filepath.Join(dir, "boots-video-horizontal.mp4"),
			want:    "16:9",
			wantErr: false,
		},
		{
			name:    "Vertical",
			path:    filepath.Join(dir, "boots-video-vertical.mp4"),
			want:    "9:16",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := getVideoAspectRatio(tt.path)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("getVideoAspectRatio() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("getVideoAspectRatio() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("getVideoAspectRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}
