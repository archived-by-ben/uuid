package archive

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"testing"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func TestExtractArchive(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExtractArchive(tt.args.name)
		})
	}
}

func BenchmarkCalculate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ExtractArchive("/Users/ben/Downloads/Miitopia_EUR_MULTi6-TRSI.zip")
	}
}
