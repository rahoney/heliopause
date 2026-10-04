package promotion

// GoProjectPromotion publishes a frozen, approved update under an original
// project guard. It has no package-manager execution capability.
type GoProjectPromotion struct {
	cache     *GoVerifiedCache
	stateRoot string
}
