package cli

import "testing"

func TestPipxRunCLIParsesLiteralBoundary(t *testing.T) {
	testPythonRunCLIParsesNativeAliasesAndLiteralBoundary(t, true)
}
func TestPipxRunCLIRejectsUnsupportedBeforeRunner(t *testing.T) {
	testPythonRunCLIRejectsUnsupportedBeforeRunner(t, true)
}
func TestPipxRunCLIStaticHelpHasNoInstallTarget(t *testing.T) {
	testPythonRunCLIStaticHelpHasNoExecutionOrInstallTarget(t, true)
}
