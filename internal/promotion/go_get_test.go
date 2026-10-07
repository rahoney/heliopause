package promotion

import (
	"testing"
)

func TestGoProjectPromotionRequiresApprovalBoundaries(t *testing.T) {
	if p, err := NewGoProjectPromotion(nil, t.TempDir()); err == nil || p != nil {
		t.Fatal("raw execution boundary configured project Promotion")
	}
}
