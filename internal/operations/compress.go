package operations

import (
	"fmt"
	"io"
	"os"
	"strings"

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
	encoder, err := zstd.NewWriter(outFile)
	if err != nil {
		return "", fmt.Errorf("failed to create Zstandard writer: %w", err)
	}
	defer encoder.Close()
	// Copy the input file to the Zstandard writer
	if _, err := io.Copy(encoder, inFile); err != nil {
		return "", fmt.Errorf("failed to compress file: %w", err)
	}
	defer encoder.Close()

	if err := os.Remove(inputPath); err != nil {
		return "", fmt.Errorf("failed to remove original file: %w", err)
	}

	return outputPath, nil
}

func DecompressZstd(inputPath string) (string, error) {
	// 2) Prepare output path (strip “.zst”)
	outputPath := strings.TrimSuffix(inputPath, ".zst")

	// open the input file
	in, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", inputPath, err)
	}
	defer in.Close()

	// 1) NewReader wraps the compressed stream
	decoder, err := zstd.NewReader(in)
	if err != nil {
		return "", fmt.Errorf("zstd.NewReader: %w", err)
	}
	defer decoder.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", outputPath, err)
	}
	defer out.Close()

	// 3) Stream copy
	if _, err := io.Copy(out, decoder); err != nil {
		return "", fmt.Errorf("copy from decoder: %w", err)
	}

	return outputPath, nil
}
