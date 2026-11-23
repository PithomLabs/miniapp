// Command determinism-lint runs the DBOS workflow determinism
// analyzer over Class D packages. CI strict mode fails the build on
// any finding before a Class D production release.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/PithomLabs/miniapp/analyzer/determinism"
)

func main() {
	multichecker.Main(determinism.Analyzer)
}
