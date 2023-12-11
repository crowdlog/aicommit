package main

import (
	"aicommit/internal/data"
	"aicommit/python_src"
	"io"
	"path/filepath"

	"github.com/michaelangeloio/go-embed-python/embed_util"
	"github.com/michaelangeloio/go-embed-python/python"
)

func main2() {

	ep, err := python.NewEmbeddedPython("tmp")
	if err != nil {
		panic(err)
	}
	tiktokenLib, err := embed_util.NewEmbeddedFiles(data.Data, "tmp2")
	if err != nil {
		panic(err)
	}
	rendererSrc, err := embed_util.NewEmbeddedFiles(python_src.RendererSource, "tmp3")
	if err != nil {
		panic(err)
	}

	ep.AddPythonPath(tiktokenLib.GetExtractedPath())

	// diffString := loadDiffFileAsString()
	// println(diffString)

	args := []string{filepath.Join(rendererSrc.GetExtractedPath(), "main.py")}

	cmd := ep.PythonCmd(args...)

	// Create a stdout pipe before starting the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	panic(err)
	// }

	// Start the command
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// // Read from the stderr pipe
	// // Use io.ReadAll or similar to read the entire output
	// stderrOutput, err := io.ReadAll(stderr)
	// if err != nil {
	// 	panic(err)
	// }

	// stderrString := string(stderrOutput)
	// println(stderrString)

	// Read from the stdout pipe
	// Use io.ReadAll or similar to read the entire output
	output, err := io.ReadAll(stdout)
	if err != nil {
		panic(err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		panic(err)
	}

	// Convert the output to a string and use it
	stdoutString := string(output)
	println(stdoutString)
}
