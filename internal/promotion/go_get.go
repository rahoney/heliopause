package promotion

// GoProjectPromotion publishes a frozen, approved update under an original
// project guard. It has no package-manager execution capability.
type GoProjectPromotion struct {
	cache           *GoVerifiedCache
	stateRoot       string
	buildCheckpoint func(string) error
}

func (p *GoProjectPromotion) buildCheck(phase string) error {
	if p.buildCheckpoint != nil {
		return p.buildCheckpoint(phase)
	}
	return nil
}
