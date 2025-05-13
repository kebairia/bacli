package operations

import (
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

func CompressZstd(inputPath string) (string, error) {
	outputPath := inputPath + ".zst"

	inFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}

	defer outFile.Close()
	// Create a Zstandard writer
	writer, err := zstd.NewWriter(outFile)
	if err != nil {
		return "", fmt.Errorf("failed to create Zstandard writer: %w", err)
	}
	defer writer.Close()
	// Copy the input file to the Zstandard writer
	if _, err := io.Copy(writer, inFile); err != nil {
		return "", fmt.Errorf("failed to compress file: %w", err)
	}
	defer writer.Close()

	if err := os.Remove(inputPath); err != nil {
		return "", fmt.Errorf("failed to remove original file: %w", err)
	}

	return outputPath, nil
}
