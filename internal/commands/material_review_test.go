package commands

import "testing"

func TestNormalizeMaterialReviewAcceptsStrongEvidence(t *testing.T) {
	review := normalizeMaterialReview(map[string]any{
		"review": map[string]any{
			"usable":               true,
			"relevance_score":      float64(9),
			"readability_score":    float64(8),
			"visual_quality_score": float64(8),
			"credibility_score":    float64(7),
			"issues":               []any{},
		},
	}, "evidence")

	if review["usable"] != true || review["decision"] != "accept" {
		t.Fatalf("expected accepted evidence, got %#v", review)
	}
}

func TestNormalizeMaterialReviewRejectsBlockingIssue(t *testing.T) {
	review := normalizeMaterialReview(map[string]any{
		"review": map[string]any{
			"usable":               true,
			"relevance_score":      float64(9),
			"readability_score":    float64(9),
			"visual_quality_score": float64(9),
			"credibility_score":    float64(9),
			"issues":               []any{"obstruction"},
		},
	}, "evidence")

	if review["usable"] != false || review["decision"] != "reject" {
		t.Fatalf("expected blocking issue rejection, got %#v", review)
	}
}

func TestNormalizeMaterialReviewRejectsWeakEvidenceCredibility(t *testing.T) {
	review := normalizeMaterialReview(map[string]any{
		"review": map[string]any{
			"usable":               true,
			"relevance_score":      float64(9),
			"readability_score":    float64(8),
			"visual_quality_score": float64(8),
			"credibility_score":    float64(3),
			"issues":               []any{"uncertain_source"},
		},
	}, "evidence")

	if review["usable"] != false {
		t.Fatalf("expected low credibility rejection, got %#v", review)
	}
}
