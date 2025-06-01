package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/doraemonkeys/doraemon"
)

const defaultThreshold int64 = 100

func main() {
	// Define command-line flags
	inputFile := flag.String("i", "", "Input file path (required)")
	outputFile := flag.String("o", "", "Output file path (optional, prints to stdout if not provided)")
	thresholdVal := flag.Int64("t", defaultThreshold, fmt.Sprintf("Process numbers strictly greater than this threshold (default %d)", defaultThreshold))

	flag.Parse()

	// Validate required input file flag
	if *inputFile == "" {
		log.Println("Error: Input file path (-i) is required.")
		flag.Usage() // Print usage information
		os.Exit(1)   // Exit with an error code
	}
	filePath := *inputFile

	// Convert threshold to big.Int
	thresholdBigInt := big.NewInt(*thresholdVal)

	// Read input file content
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", filePath, err)
	}
	content := string(contentBytes)

	// Compile the regex: (?<!\d|[a-z]|[A-Z])(\d{3,})(?!\d|[a-z]|[A-Z])
	// This finds standalone numbers of 3 or more digits.
	re, err := regexp2.Compile(`(?<!\d|[a-z]|[A-Z])(\d{3,})(?!\d|[a-z]|[A-Z])`, regexp2.ECMAScript)
	if err != nil {
		log.Fatalf("Failed to compile regex: %v", err)
	}

	var resultBuilder strings.Builder
	currentIndex := 0 // Tracks the end of the last processed part

	match, _ := re.FindStringMatch(content)
	for match != nil {
		// Group 0 is the entire match. Group 1 is the captured number string `(\d{3,})`.
		// For this regex, match.String() and match.Groups()[1].String() are the same.
		numStr := match.Groups()[1].String()

		// Append the part of the content string before the current match
		resultBuilder.WriteString(content[currentIndex:match.Index])

		bigNum, parseOk := new(big.Int).SetString(numStr, 10)
		if !parseOk {
			// This should ideally not happen with a \d{3,} regex.
			log.Printf("Warning: Could not parse '%s' as a number. Writing original: \"%s\"", numStr, match.String())
			resultBuilder.WriteString(match.String()) // Write the original full match
		} else {
			replaced := false
			// Process only if the number is strictly greater than the threshold
			if bigNum.Cmp(thresholdBigInt) > 0 {
				// Try (2^n - 1) << m
				canFormatMinusOne, formattedStrMinusOne := doraemon.FormatAsPowerOfTwoMinusOneShiftedBig(bigNum)
				if canFormatMinusOne {
					resultBuilder.WriteString(formattedStrMinusOne)
					replaced = true
				} else {
					// If not replaced, try (2^n + 1) << m
					canFormatPlusOne, formattedStrPlusOne := doraemon.FormatAsPowerOfTwoPlusOneShiftedBig(bigNum)
					if canFormatPlusOne {
						resultBuilder.WriteString(formattedStrPlusOne)
						replaced = true
					}
				}
			}

			if !replaced {
				resultBuilder.WriteString(match.String()) // Write original number if no replacement or not over threshold
			}
		}

		currentIndex = match.Index + match.Length
		match, _ = re.FindNextMatch(match)
	}

	// Append the rest of the content string after the last match (or the whole string if no matches)
	resultBuilder.WriteString(content[currentIndex:])

	// Determine output destination and write the result
	var out io.Writer = os.Stdout // Default to standard output
	if *outputFile != "" {
		file, err := os.Create(*outputFile) // Create or truncate the output file
		if err != nil {
			log.Fatalf("Failed to create output file %s: %v", *outputFile, err)
		}
		defer file.Close()
		out = file
	}

	_, err = fmt.Fprint(out, resultBuilder.String())
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	// Log success if writing to a file
	if *outputFile != "" {
		log.Printf("Successfully processed %s and wrote output to %s", *inputFile, *outputFile)
	}
}
