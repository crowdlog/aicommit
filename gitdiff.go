package main

import (
	"bufio"
	"strings"
)

// Chunk represents a portion of the git diff
type Chunk struct {
	Lines []string
}

// SplitGitDiff splits a git diff into chunks based on file changes and a specified length
func SplitGitDiff(diff string, length int) [][]Chunk {
	var chunks [][]Chunk
	var currentChunk []Chunk
	var currentLines []string

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for a new file diff start
		if strings.HasPrefix(line, "diff --git") {
			if len(currentLines) > 0 {
				currentChunk = append(currentChunk, Chunk{Lines: currentLines})
				currentLines = []string{}
			}

			if len(currentChunk) > 0 {
				chunks = append(chunks, currentChunk)
				currentChunk = []Chunk{}
			}
		}

		currentLines = append(currentLines, line)

		// Check if current file diff exceeds the length limit
		if len(currentLines) >= length {
			currentChunk = append(currentChunk, Chunk{Lines: currentLines})
			currentLines = []string{}
		}
	}

	// Add the last chunk
	if len(currentLines) > 0 {
		currentChunk = append(currentChunk, Chunk{Lines: currentLines})
	}
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// func main() {
// 	// Example usage
// 	diff := `diff --git a/file1.txt b/file1.txt
// index 1234567..89abcde 100644
// --- a/file1.txt
// +++ b/file1.txt
// @@ -1,3 +1,3 @@
// -Hello
// +Hello, World, 2, 100
//  Bye
// diff --git a/file2.txt b/file2.txt
// index 2345678..9abcdef 100644
// --- a/file2.txt
// +++ b/file2.txt
// @@ -1,3 +1,3 @@
// -Goodbye
// +Goodbye, World
//  Hello`
// 	diffTimesX := strings.Repeat(diff, 1000) // 1000 times the diff

// 	chunks := SplitGitDiff(diffTimesX, 10) // Splitting the diff into chunks of 10 lines

// 	for _, chunk := range chunks {
// 		for _, subChunk := range chunk {
// 			fmt.Println("Chunk:")
// 			for _, line := range subChunk.Lines {
// 				fmt.Println(line)
// 			}
// 			fmt.Println()
// 		}
// 	}

// 	tkm, err := tiktoken.GetEncoding("cl100k_base")
// 	if err != nil {
// 		panic(err)
// 	}
// 	token := tkm.Encode(diffTimesX, nil, nil)
// 	fmt.Println(len(token), "tokens")

// }
