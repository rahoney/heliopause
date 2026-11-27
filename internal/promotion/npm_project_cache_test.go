package promotion

import (
	"github.com/rahoney/heliopause/internal/core/domain"
	"testing"
)

func TestNPMProjectControlsBindFactsAcrossDomainSortOrder(t *testing.T) {
	manifest, err := domain.NewProjectControlFile("package.json", []byte(`{"dependencies":{}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := domain.NewProjectControlFile("package-lock.json", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	owner := &PNPMProjectPromotion{client: npmProjectCacheNPM}
	original := []domain.ProjectControlFile{manifest, lock}
	canonical := []domain.ProjectControlFile{lock, manifest}
	if !owner.sameControls(original, canonical) {
		t.Fatal("domain-sorted controls lost their original binding")
	}
	changed, err := domain.NewProjectControlFile("package.json", []byte(`{"dependencies":{"is-number":"7.0.0"}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	presentLock, err := domain.NewProjectControlFile("package-lock.json", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range [][]domain.ProjectControlFile{{lock, changed}, {presentLock, manifest}, {lock, lock}, {manifest}, {}} {
		if owner.sameControls(original, candidate) {
			t.Fatal("changed or partial npm control binding accepted")
		}
	}
}
