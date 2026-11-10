package pypi

// ValidatePythonExecutionStartup reuses the existing complete-wheel startup
// restriction for an isolated console lane. This is DATA validation only;
// it grants no inspection coverage, Policy or execution admission.
func ValidatePythonExecutionStartup(wheel WheelInspection) error {
	return validatePipxMetadataSurface(wheel)
}
