package matching

import (
	"context"
	"errors"
	"fmt"

	"fp-manager/internal/scanner"
	"fp-manager/internal/storage"
)

const (
	// DefaultVerifyThreshold is Mantra's recommended 1:1 match threshold (approx 90% confidence).
	DefaultVerifyThreshold = 1400

	// DefaultIdentifyThreshold is a slightly higher threshold for 1:N to minimize false accepts.
	DefaultIdentifyThreshold = 1800
)

// VerifyResult contains the outcome of a 1:1 verification attempt.
type VerifyResult struct {
	Matched       bool   `json:"matched"`
	Score         int    `json:"score"`
	UserID        string `json:"user_id"`
	UserName      string `json:"user_name"`
	MatchedFinger string `json:"matched_finger,omitempty"`
}

// IdentifyResult contains the outcome of a 1:N identification attempt.
type IdentifyResult struct {
	Identified    bool          `json:"identified"`
	Score         int           `json:"score"`
	User          *storage.User `json:"user,omitempty"`
	MatchedFinger string        `json:"matched_finger,omitempty"`
	Ambiguous     bool          `json:"ambiguous"`
}

type Service struct {
	sc scanner.Scanner
	db *storage.DB
}

func NewService(sc scanner.Scanner, db *storage.DB) *Service {
	return &Service{
		sc: sc,
		db: db,
	}
}

// VerifyUser performs 1:1 verification of a probe template against a specific user's enrolled templates.
func (s *Service) VerifyUser(ctx context.Context, userID string, probeISO []byte) (*VerifyResult, error) {
	if len(probeISO) == 0 {
		return nil, errors.New("probe template cannot be empty")
	}

	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	templates, err := s.db.GetUserTemplates(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user templates: %w", err)
	}

	if len(templates) == 0 {
		return nil, errors.New("user has no enrolled fingerprints")
	}

	bestScore := 0
	matchedFinger := ""
	isMatched := false

	for _, tmpl := range templates {
		score, match, err := s.sc.MatchISO(probeISO, tmpl.DecryptedBytes)
		if err != nil {
			continue
		}
		if score > bestScore {
			bestScore = score
			matchedFinger = tmpl.FingerLabel
		}
		if match || score >= DefaultVerifyThreshold {
			isMatched = true
		}
	}

	result := &VerifyResult{
		Matched:       isMatched,
		Score:         bestScore,
		UserID:        user.ID,
		UserName:      user.DisplayName,
		MatchedFinger: matchedFinger,
	}

	if isMatched {
		_ = s.db.LogAudit(ctx, "VERIFY_SUCCESS", user.ID, user.DisplayName, "SUCCESS", bestScore,
			fmt.Sprintf("Matched finger: %s with score: %d", matchedFinger, bestScore))
	} else {
		_ = s.db.LogAudit(ctx, "VERIFY_FAILED", user.ID, user.DisplayName, "FAILURE", bestScore,
			fmt.Sprintf("Verification failed, highest score: %d", bestScore))
	}

	return result, nil
}

// IdentifyUser searches all enrolled templates (1:N) to identify the person.
func (s *Service) IdentifyUser(ctx context.Context, probeISO []byte) (*IdentifyResult, error) {
	if len(probeISO) == 0 {
		return nil, errors.New("probe template cannot be empty")
	}

	allTemplates, err := s.db.GetAllTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}

	if len(allTemplates) == 0 {
		return nil, errors.New("no enrolled templates in system")
	}

	type matchCandidate struct {
		userID      string
		fingerLabel string
		score       int
	}

	var candidates []matchCandidate

	for _, tmpl := range allTemplates {
		score, _, err := s.sc.MatchISO(probeISO, tmpl.DecryptedBytes)
		if err != nil {
			continue
		}
		if score >= DefaultIdentifyThreshold {
			candidates = append(candidates, matchCandidate{
				userID:      tmpl.UserID,
				fingerLabel: tmpl.FingerLabel,
				score:       score,
			})
		}
	}

	if len(candidates) == 0 {
		_ = s.db.LogAudit(ctx, "IDENTIFY_FAILED", "", "", "FAILURE", 0, "No enrolled user matched probe template")
		return &IdentifyResult{Identified: false}, nil
	}

	// Find top candidate
	top := candidates[0]
	ambiguous := false

	for _, c := range candidates[1:] {
		if c.score > top.score {
			top = c
		} else if c.score == top.score && c.userID != top.userID {
			ambiguous = true
		}
	}

	if ambiguous {
		_ = s.db.LogAudit(ctx, "IDENTIFY_AMBIGUOUS", "", "", "AMBIGUOUS", top.score, "Multiple distinct users matched with identical high score")
		return &IdentifyResult{Identified: false, Ambiguous: true}, nil
	}

	user, err := s.db.GetUser(ctx, top.userID)
	if err != nil {
		return nil, err
	}

	_ = s.db.LogAudit(ctx, "IDENTIFY_SUCCESS", user.ID, user.DisplayName, "SUCCESS", top.score,
		fmt.Sprintf("Identified user %s via %s (score %d)", user.DisplayName, top.fingerLabel, top.score))

	return &IdentifyResult{
		Identified:    true,
		Score:         top.score,
		User:          user,
		MatchedFinger: top.fingerLabel,
	}, nil
}
