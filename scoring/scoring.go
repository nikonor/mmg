package scoring

import (
	"fmt"

	"mmg-tournament/models"
)

// CalculateCoefficients calculates Berger and Buchholz coefficients for all players
// This should be called after all rounds have been played and points updated
func CalculateCoefficients(players []*models.Player, roundNum int) {
	// Create a map for quick lookup of players by ID
	playerMap := make(map[int]*models.Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	// Calculate total points for each player (MMS + earned)
	// First, we need to determine total McMahon groups
	maxGroup := 0
	for _, p := range players {
		if p.McMahonGroup > maxGroup {
			maxGroup = p.McMahonGroup
		}
	}

	// Update each player's total points
	for _, p := range players {
		mms := models.CalculateMCS(p.McMahonGroup, maxGroup)
		earned := p.GetPointsEarned()
		p.Points = mms + earned
	}

	// Calculate coefficients for each player
	for _, player := range players {
		berger := 0.0
		buchholz := 0.0

		// Look at all played rounds up to roundNum
		for i := 0; i < roundNum && i < len(player.Results); i++ {
			result := player.Results[i]

			// Skip bye rounds
			if result.IsBye {
				continue
			}

			// Skip unplayed rounds
			if result.OpponentID == 0 {
				continue
			}

			// Get opponent
			opponent, exists := playerMap[result.OpponentID]
			if !exists {
				continue
			}

			// Calculate opponent's total points
			opponentMMS := models.CalculateMCS(opponent.McMahonGroup, maxGroup)
			opponentEarned := opponent.GetPointsEarned()
			opponentPoints := float64(opponentMMS + opponentEarned)

			// Buchholz: sum of all opponents' points
			buchholz += opponentPoints

			// Berger: sum of defeated opponents' points
			if result.IsWin() {
				berger += opponentPoints
			}
		}

		player.Berger = berger
		player.Buchholz = buchholz
	}
}

// AssignMcMahonGroups assigns McMahon groups to players sorted by rating
// Players are divided into numGroups equal-sized groups by rating
// If not evenly divisible, the last (weakest) groups may have fewer players
func AssignMcMahonGroups(players []*models.Player, numGroups int) {
	// Sort by rating (should already be sorted, but ensure)
	models.SortPlayersByRating(players)

	totalPlayers := len(players)

	// According to TZ example: 21 players / 4 groups = 6,6,6,3
	// So the first 'extra' groups get baseSize + 1, rest get baseSize
	// But 21/4 = 5 remainder 1, which would give 6,5,5,5
	// Looking at the example more carefully: they seem to aim for groups of ~6
	// Let me recalculate: ceil(21/4) = 6, so we aim for groups of 6, and last group gets remainder

	// Actually, the TZ says "равных по количеству участников групп"
	// but example shows 6,6,6,3. Let's use a different approach:
	// Calculate ideal size rounding up, and last group gets whatever is left

	idealSize := (totalPlayers + numGroups - 1) / numGroups // ceiling division

	currentIdx := 0
	for group := 1; group <= numGroups; group++ {
		groupSize := idealSize
		remaining := totalPlayers - currentIdx
		if remaining < idealSize {
			groupSize = remaining
		}

		for i := 0; i < groupSize && currentIdx < totalPlayers; i++ {
			players[currentIdx].McMahonGroup = group
			currentIdx++
		}
	}
}

// CheckAllResultsPlayed verifies that all round results for all players have been entered.
// It returns an error if any round result is missing or unplayed (Win == nil and not a bye).
func CheckAllResultsPlayed(players []*models.Player, totalRounds int) error {
	for _, p := range players {
		if len(p.Results) < totalRounds {
			return fmt.Errorf("игрок %s (ID=%d): недостаточно результатов (ожидается %d, имеется %d)",
				p.Name, p.ID, totalRounds, len(p.Results))
		}
		for i := 0; i < totalRounds; i++ {
			r := p.Results[i]
			// Bye counts as played
			if r.IsBye {
				continue
			}
			// Check if result is unplayed
			if !r.HasPlayed() {
				return fmt.Errorf("игрок %s (ID=%d): тур %d не сыгран (результат: %s)",
					p.Name, p.ID, i+1, r.String())
			}
		}
	}
	return nil
}

// FinalizeCoefficients recalculates points and coefficients after ALL rounds are complete.
// This is similar to CalculateCoefficients but ensures final accurate state.
func FinalizeCoefficients(players []*models.Player) {
	// Reuse CalculateCoefficients with all rounds
	roundNum := len(players[0].Results)
	if roundNum > 0 {
		CalculateCoefficients(players, roundNum)
	}
}

// AssignIDs assigns unique IDs to players starting from 1
func AssignIDs(players []*models.Player) {
	for i, p := range players {
		p.ID = i + 1
	}
}

// UpdatePlaces updates the place field based on current sorting
func UpdatePlaces(players []*models.Player) {
	for i, p := range players {
		p.Place = i + 1
	}
}
