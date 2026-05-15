package core

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func (b *Backend) Recommend(opts RecommendOptions) ([]RecommendRow, string, error) {
	projectDir, err := b.ProjectDir(opts.Project)
	if err != nil {
		return nil, "", err
	}
	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		return nil, "", fmt.Errorf("project directory does not exist: %s", projectDir)
	}
	skills, warning, err := b.ListSkills("")
	if err != nil {
		return nil, warning, err
	}
	return newDefaultRecommendEngine().recommend(projectDir, skills), warning, nil
}

type recommendEngine struct {
	providers []signalProvider
}

func newDefaultRecommendEngine() recommendEngine {
	return newRecommendEngine(defaultSignalProviders())
}

func newRecommendEngine(providers []signalProvider) recommendEngine {
	return recommendEngine{providers: append([]signalProvider(nil), providers...)}
}

func (e recommendEngine) recommend(projectDir string, skills []Skill) []RecommendRow {
	signals := collectProjectSignalsWithProviders(projectDir, e.providers)
	rows := []RecommendRow{}
	for _, skill := range skills {
		score := 0
		reasons := []string{}
		seenReason := map[string]bool{}
		add := func(token string) {
			token = strings.ToLower(strings.TrimSpace(token))
			if signal, ok := signals[token]; ok {
				score += signal.Weight
				if !seenReason[signal.Evidence] {
					seenReason[signal.Evidence] = true
					reasons = append(reasons, signal.Evidence)
				}
			}
		}
		add(skill.Category)
		for _, trigger := range strings.Split(skill.Triggers, ",") {
			add(trigger)
		}
		if score > 0 {
			rows = append(rows, RecommendRow{
				Source: skill.Source, Skill: skill.Name, Score: score,
				Reason:     "matched " + strings.Join(reasons, ", "),
				InstallArg: skill.Source + "/" + skill.Name,
			})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		if rows[i].Source != rows[j].Source {
			return rows[i].Source < rows[j].Source
		}
		return rows[i].Skill < rows[j].Skill
	})
	return rows
}
