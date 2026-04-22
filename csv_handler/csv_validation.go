package csv_handler

import (
	"fmt"

	"mmg-tournament/models"
)

// ValidationResult contains the results of the validation
type ValidationResult struct {
	IsValid  bool
	Errors   []string
	Warnings []string
}

// ValidateTournamentData validates that tournament results up to roundNum are complete and consistent
// Checks:
// 1. All results for rounds 1..roundNum are entered (have opponent and result)
// 2. For each match, both players have consistent results (opposite wins)
func ValidateTournamentData(players []*models.Player, roundNum int) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check that we have players
	if len(players) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "no players found")
		return result
	}

	// Check that roundNum is valid
	if roundNum < 1 {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid roundNum: %d", roundNum))
		return result
	}

	// Build a map of player ID to player for quick lookup
	playerMap := make(map[int]*models.Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	// Validate each player has results for rounds 1 to roundNum
	for _, player := range players {
		// Check each round up to roundNum
		for roundIdx := 0; roundIdx < roundNum && roundIdx < len(player.Results); roundIdx++ {
			res := player.Results[roundIdx]
			roundNumDisplay := roundIdx + 1

			// Skip bye checks
			if res.IsBye {
				continue
			}

			// Check if the result is complete (has opponent and result)
			if res.OpponentID == 0 || res.Win == nil {
				result.IsValid = false
				if res.OpponentID == 0 {
					result.Errors = append(result.Errors,
						fmt.Sprintf("player %s, round %d: opponent ID is missing",
							player.Name, roundNumDisplay))
				}
				if res.Win == nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("player %s, round %d: result is missing",
							player.Name, roundNumDisplay))
				}
				continue
			}

			// Check consistency with opponent's result
			if opponent, exists := playerMap[res.OpponentID]; exists {
				if roundIdx < len(opponent.Results) {
					opponentRes := opponent.Results[roundIdx]

					// If opponent has no result yet, it's a warning
					if opponentRes.OpponentID == 0 || opponentRes.Win == nil {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("player %s, round %d: opponent %s has no result for this match",
								player.Name, roundNumDisplay, opponent.Name))
						continue
					}

					// Check that opponents match
					if opponentRes.OpponentID != player.ID {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("player %s, round %d: opponent ID mismatch - player claims opponent is %d, but opponent's record shows %d",
								player.Name, roundNumDisplay, res.OpponentID, opponentRes.OpponentID))
					}

					// Check that results are opposite (one win, one loss)
					if *res.Win == *opponentRes.Win {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("player %s, round %d: both players have the same result for match against %s",
								player.Name, roundNumDisplay, opponent.Name))
					}
				}
			} else {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("player %s, round %d: opponent with ID %d not found in player list",
						player.Name, roundNumDisplay, res.OpponentID))
			}
		}
		// If player has fewer results than needed, it's an error
		if roundNum > len(player.Results) {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("player %s has only %d results but needs %d rounds",
					player.Name, len(player.Results), roundNum))
		}
	}

	return result
}

// ErrorString returns a formatted error message string
func (v *ValidationResult) ErrorString() string {
	if v.IsValid {
		return ""
	}

	parts := []string{}

	if len(v.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("Errors (%d):", len(v.Errors)))
		for _, err := range v.Errors {
			parts = append(parts, fmt.Sprintf("  - %s", err))
		}
	}

	if len(v.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("Warnings (%d):", len(v.Warnings)))
		for _, warn := range v.Warnings {
			parts = append(parts, fmt.Sprintf("  - %s", warn))
		}
	}

	return "\n" + joinStrings(parts, "\n")
}

// joinStrings joins strings with separator, skipping empty ones
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 && result != "" {
			result += sep
		}
		result += p
	}
	return result
}
