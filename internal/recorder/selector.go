package recorder

import "web-automation/internal/models"

// RankSelectors orders selector candidates by stability score.
func RankSelectors(candidates []models.SelectorCandidate) []models.SelectorCandidate {
	sorted := make([]models.SelectorCandidate, len(candidates))
	copy(sorted, candidates)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Score > sorted[i].Score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

// BestSelector returns the highest-ranked selector string.
func BestSelector(candidates []models.SelectorCandidate) string {
	if len(candidates) == 0 {
		return ""
	}
	ranked := RankSelectors(candidates)
	return ranked[0].Selector
}
