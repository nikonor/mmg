package csv_handler

import (
	"strings"
	"testing"

	"mmg-tournament/models"
)

func TestValidateTournamentData_NoPlayers(t *testing.T) {
	result := ValidateTournamentData([]*models.Player{}, 3)
	if result.IsValid {
		t.Error("Expected validation to fail with no players")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error for no players")
	}
}

func TestValidateTournamentData_InvalidTotalRounds(t *testing.T) {
	players := []*models.Player{
		{ID: 1, Name: "Player1", Results: []models.RoundResult{}},
	}
	result := ValidateTournamentData(players, 0)
	if result.IsValid {
		t.Error("Expected validation to fail with invalid totalRounds")
	}
}

func TestValidateTournamentData_ValidData(t *testing.T) {
	// Create two players with consistent results
	result1 := true
	result2 := false
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: &result1}, // Alice beats Bob
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: &result2}, // Bob loses to Alice
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if !result.IsValid {
		t.Errorf("Expected valid data, got errors: %v", result.Errors)
	}
}

func TestValidateTournamentData_BothPlayersSameResult(t *testing.T) {
	result1 := true
	result2 := true
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: &result1}, // Alice says she won
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: &result2}, // Bob also says he won
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if result.IsValid {
		t.Error("Expected validation to fail when both players have same result")
	}

	foundDuplicate := false
	for _, err := range result.Errors {
		if contains(err, "same result") {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Error("Expected error message about both players having same result")
	}
}

func TestValidateTournamentData_MissingOpponent(t *testing.T) {
	result1 := true
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 999, Win: &result1}, // Opponent 999 doesn't exist
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: new(bool)},
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if result.IsValid {
		t.Error("Expected validation to fail when opponent not found")
	}

	foundMissingOpponent := false
	for _, err := range result.Errors {
		if contains(err, "opponent") && contains(err, "not found") {
			foundMissingOpponent = true
			break
		}
	}
	if !foundMissingOpponent {
		t.Error("Expected error message about missing opponent")
	}
}

func TestValidateTournamentData_MissingResult(t *testing.T) {
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: nil}, // No result yet
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: new(bool)},
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if result.IsValid {
		t.Error("Expected validation to fail when result is missing")
	}
}

func TestValidateTournamentData_MissingOpponentID(t *testing.T) {
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 0, Win: new(bool)}, // No opponent ID
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: new(bool)},
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if result.IsValid {
		t.Error("Expected validation to fail when opponent ID is missing")
	}
}

func TestValidateTournamentData_NotEnoughResults(t *testing.T) {
	result1 := true
	result2 := false
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: &result1},
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: &result2},
			},
		},
	}

	// Only 1 result but totalRounds = 3
	result := ValidateTournamentData(players, 3)
	if result.IsValid {
		t.Error("Expected validation to fail when player doesn't have enough results")
	}

	foundInsufficient := false
	for _, err := range result.Errors {
		if contains(err, "has only") && contains(err, "results") {
			foundInsufficient = true
			break
		}
	}
	if !foundInsufficient {
		t.Error("Expected error message about insufficient results")
	}
}

func TestValidateTournamentData_Byes(t *testing.T) {
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 0, IsBye: true, Win: new(bool)},
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 0, IsBye: true, Win: new(bool)},
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	// Bye is allowed, this should pass
	if !result.IsValid {
		t.Errorf("Expected valid data with bye, got errors: %v", result.Errors)
	}
}

func TestValidateTournamentData_MultipleRounds(t *testing.T) {
	// Round 1: Alice (1) vs Bob (2) - Alice wins
	// Round 2: Alice (1) vs Bob (2) again - Bob wins
	r1Alice := true
	r1Bob := false
	r2Alice := false
	r2Bob := true
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: &r1Alice}, // Round 1: Alice vs Bob
				{OpponentID: 2, Win: &r2Alice}, // Round 2: Alice vs Bob
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 1, Win: &r1Bob}, // Round 1: Bob vs Alice
				{OpponentID: 1, Win: &r2Bob}, // Round 2: Bob vs Alice
			},
		},
	}

	result := ValidateTournamentData(players, 2)
	if !result.IsValid {
		t.Errorf("Expected valid data, got errors: %v", result.Errors)
	}
}

func TestValidateTournamentData_OpponentMismatch(t *testing.T) {
	result1 := true
	result2 := false
	players := []*models.Player{
		{
			ID:   1,
			Name: "Alice",
			Results: []models.RoundResult{
				{OpponentID: 2, Win: &result1}, // Alice plays Bob
			},
		},
		{
			ID:   2,
			Name: "Bob",
			Results: []models.RoundResult{
				{OpponentID: 3, Win: &result2}, // Bob says he plays Charlie
			},
		},
	}

	result := ValidateTournamentData(players, 1)
	if result.IsValid {
		t.Error("Expected validation to fail when opponent IDs don't match")
	}

	foundMismatch := false
	for _, err := range result.Errors {
		if contains(err, "opponent ID mismatch") {
			foundMismatch = true
			break
		}
	}
	if !foundMismatch {
		t.Error("Expected error about opponent ID mismatch")
	}
}

func TestValidationResult_ErrorString(t *testing.T) {
	result := ValidationResult{
		IsValid: false,
		Errors:  []string{"error1", "error2"},
	}

	errorStr := result.ErrorString()
	if errorStr == "" {
		t.Error("Expected non-empty error string")
	}
	if !contains(errorStr, "Errors (2):") {
		t.Error("Expected error count in error string")
	}
	if !contains(errorStr, "error1") {
		t.Error("Expected error1 in error string")
	}
}

func TestValidationResult_ErrorString_NoErrors(t *testing.T) {
	result := ValidationResult{
		IsValid: true,
	}

	errorStr := result.ErrorString()
	if errorStr != "" {
		t.Errorf("Expected empty error string for valid result, got: %q", errorStr)
	}
}

func TestValidationResult_ErrorString_Warnings(t *testing.T) {
	result := ValidationResult{
		IsValid:  false,
		Errors:   []string{"error1"},
		Warnings: []string{"warning1", "warning2"},
	}

	errorStr := result.ErrorString()
	if !contains(errorStr, "Warnings (2):") {
		t.Error("Expected warning count in error string")
	}
	if !contains(errorStr, "warning1") {
		t.Error("Expected warning1 in error string")
	}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
