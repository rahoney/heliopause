package sandbox

// isolatedContainerEnvironmentArguments constructs the complete Host-derived
// environment surface for every HAA container. Docker never receives the
// caller environment: image defaults are pinned runtime data and these values
// are the only controller additions. This is an allowlist, not a credential
// name blacklist.
func isolatedContainerEnvironmentArguments() []string {
	arguments := make([]string, 0, 8)
	for _, entry := range []string{
		"HOME=/tmp",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LANG=C",
		"LC_ALL=C",
	} {
		arguments = append(arguments, "--env", entry)
	}
	return arguments
}
