//go:build ignore

// Command clang-wrap is a CC wrapper for Go cgo builds on Windows.
//
// Go's cmd/go unconditionally appends -mthreads to the compiler command line
// for windows cgo builds. clang (MSVC target) rejects that flag, so this
// wrapper strips it and forwards every other argument unchanged to clang.
//
// A .cmd/.bat wrapper cannot do this job: cmd.exe splits batch arguments on
// '=', ',' and ';' (e.g. -fmessage-length=0 arrives as two args), and
// re-quoting arguments that end in a backslash corrupts them. A native
// executable receives the exact argv that Go's os/exec built, so all
// arguments pass through byte-for-byte.
package main

import (
	"fmt"
	"os"
	"strings"
)

const clangPath = `C:\Program Files\LLVM\bin\clang.exe`

func main() {
	args := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "-mthreads") {
			continue
		}
		args = append(args, arg)
	}

	// Start clang directly with no shell and no PATH lookup (clangPath is
	// absolute). argv[0] must be the program path itself, matching what
	// os/exec would have built. Stdin/stdout/stderr are inherited so clang
	// talks to the same console Go's build process uses.
	proc, err := os.StartProcess(clangPath, append([]string{clangPath}, args...), &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "clang-wrap:", err)
		os.Exit(1)
	}

	state, err := proc.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "clang-wrap:", err)
		os.Exit(1)
	}
	os.Exit(state.ExitCode())
}