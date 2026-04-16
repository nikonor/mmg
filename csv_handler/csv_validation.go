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

// ValidateTournamentData validates that all round results are consistent
// Checks:
// 1. All previous round results exist (have opponent and result)
// 2. For each match, both players have consistent results (opposite wins)
func ValidateTournamentData(players []*models.Player, totalRounds int) ValidationResult {
	result := ValidationResult{IsValid: true}

	// Check that we have players
	if len(players) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "no players found")
		return result
	}

	// Check that totalRounds is valid
	if totalRounds < 1 {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid totalRounds: %d", totalRounds))
		return result
	}

	// Build a map of player ID to player for quick lookup
	playerMap := make(map[int]*models.Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	// Validate each player has results for all rounds
	for _, player := range players {
		if len(player.Results) < totalRounds {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("player %s has only %d results but needs %d rounds",
					player.Name, len(player.Results), totalRounds))
			continue
		}

		// Check each round for this player
		for roundIdx, res := range player.Results {
			roundNum := roundIdx + 1

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
							player.Name, roundNum))
				}
				if res.Win == nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("player %s, round %d: result is missing",
							player.Name, roundNum))
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
								player.Name, roundNum, opponent.Name))
						continue
					}

					// Check that opponents match
					if opponentRes.OpponentID != player.ID {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("player %s, round %d: opponent ID mismatch - player claims opponent is %d, but opponent's record shows %d",
								player.Name, roundNum, res.OpponentID, opponentRes.OpponentID))
					}

					// Check that results are opposite (one win, one loss)
					if *res.Win == *opponentRes.Win {
						result.IsValid = false
						result.Errors = append(result.Errors,
							fmt.Sprintf("player %s, round %d: both players have the same result for match against %s",
								player.Name, roundNum, opponent.Name))
					}
				}
			} else {
				result.IsValid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("player %s, round %d: opponent with ID %d not found in player list",
						player.Name, roundNum, res.OpponentID))
			}
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
