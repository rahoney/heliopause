package domain

// CapturedProcessOutput is bounded private-memory transport DATA, never guest
// target/exit, inspection coverage or Policy authority. Accessors return copies;
// implementations keep raw bytes out of ordinary formatting and persistence.
type CapturedProcessOutput interface {
	Stdout() []byte
	Stderr() []byte
	StdoutOverflow() bool
	StderrOverflow() bool
}
