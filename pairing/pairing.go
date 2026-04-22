package pairing

import (
	"sort"

	"mmg-tournament/models"
)

// MakePairs creates pairs for the current round according to McMahon/Swiss system rules
// For round 1: pair within McMahon groups
// For subsequent rounds: pair based on current points
func MakePairs(players []*models.Player, roundNum int) {
	if roundNum == 1 {
		makePairsRound1(players, roundNum)
		return
	}
	makePairsSubsequent(players, roundNum)
}

// makePairsRound1 pairs players within their McMahon groups for the first round
func makePairsRound1(players []*models.Player, roundNum int) {
	paired := make(map[int]bool)

	// Group by McMahon group
	mcmahonGroups := make(map[int][]*models.Player)
	groupOrder := make([]int, 0)

	for _, p := range players {
		if _, exists := mcmahonGroups[p.McMahonGroup]; !exists {
			groupOrder = append(groupOrder, p.McMahonGroup)
		}
		mcmahonGroups[p.McMahonGroup] = append(mcmahonGroups[p.McMahonGroup], p)
	}

	// Process each McMahon group
	for _, groupNum := range groupOrder {
		groupPlayers := mcmahonGroups[groupNum]

		// Collect unpaired candidates
		var candidates []*models.Player
		for _, p := range groupPlayers {
			if !paired[p.ID] {
				candidates = append(candidates, p)
			}
		}

		// Sort by number of available opponents (least options first)
		sort.Slice(candidates, func(i, j int) bool {
			availI := countAvailableOpponents(candidates[i], candidates, paired)
			availJ := countAvailableOpponents(candidates[j], candidates, paired)
			if availI == availJ {
				return candidates[i].ID < candidates[j].ID // stable sort
			}
			return availI < availJ
		})

		// Try to pair each player in order of difficulty
		for _, player := range candidates {
			if paired[player.ID] {
				continue
			}

			opponent := findBestOpponent(player, candidates, paired)
			if opponent != nil {
				setPairing(player, opponent, roundNum)
				paired[player.ID] = true
				paired[opponent.ID] = true
			} else {
				setBye(player, roundNum)
				paired[player.ID] = true
			}
		}
	}
}

// makePairsSubsequent pairs players based on current points for rounds 2+
func makePairsSubsequent(players []*models.Player, roundNum int) {
	paired := make(map[int]bool)

	// Group by points
	pointsMap := make(map[int][]*models.Player)
	for _, p := range players {
		pointsMap[p.Points] = append(pointsMap[p.Points], p)
	}

	// Sort point groups in descending order
	var pointsVals []int
	for pts := range pointsMap {
		pointsVals = append(pointsVals, pts)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(pointsVals)))

	// First pass: maximize internal pairings in each group
	for _, pts := range pointsVals {
		groupPlayers := pointsMap[pts]
		pairWithinGroup(groupPlayers, paired, roundNum)
	}

	// Second pass: handle leftovers by pairing with lower-score groups
	for i, pts := range pointsVals {
		var unpairedInGroup []*models.Player
		for _, p := range pointsMap[pts] {
			if !paired[p.ID] {
				unpairedInGroup = append(unpairedInGroup, p)
			}
		}

		for _, player := range unpairedInGroup {
			if paired[player.ID] {
				continue
			}

			var opponent *models.Player
			// Search only in lower groups (i+1 and below)
			for j := i + 1; j < len(pointsVals); j++ {
				lowerGroup := pointsMap[pointsVals[j]]
				opponent = findBestOpponent(player, lowerGroup, paired)
				if opponent != nil {
					break
				}
			}

			if opponent != nil {
				setPairing(player, opponent, roundNum)
				paired[player.ID] = true
				paired[opponent.ID] = true
			} else {
				setBye(player, roundNum)
				paired[player.ID] = true
			}
		}
	}
}

// pairWithinGroup tries to pair as many players as possible within the same score group
// Uses difficulty-based sorting: players with fewer possible opponents go first
func pairWithinGroup(groupPlayers []*models.Player, paired map[int]bool, roundNum int) {
	var candidates []*models.Player
	for _, p := range groupPlayers {
		if !paired[p.ID] {
			candidates = append(candidates, p)
		}
	}

	// Sort by number of available opponents (least options first)
	sort.Slice(candidates, func(i, j int) bool {
		availI := countAvailableOpponents(candidates[i], candidates, paired)
		availJ := countAvailableOpponents(candidates[j], candidates, paired)
		if availI == availJ {
			return candidates[i].ID < candidates[j].ID // stable sort
		}
		return availI < availJ
	})

	// Try to pair each player in order of difficulty
	for _, player := range candidates {
		if paired[player.ID] {
			continue
		}

		opponent := findBestOpponent(player, candidates, paired)
		if opponent != nil {
			setPairing(player, opponent, roundNum)
			paired[player.ID] = true
			paired[opponent.ID] = true
		}
	}
}

// countAvailableOpponents returns how many unpaired players this player can face
func countAvailableOpponents(player *models.Player, candidates []*models.Player, paired map[int]bool) int {
	count := 0
	for _, c := range candidates {
		if paired[c.ID] || c.ID == player.ID {
			continue
		}
		if !havePlayed(player, c) {
			count++
		}
	}
	return count
}

// findBestOpponent finds an opponent who hasn't played against the player
// Prefers players with fewer remaining options (to avoid stranding them)
func findBestOpponent(player *models.Player, candidates []*models.Player, paired map[int]bool) *models.Player {
	var best *models.Player
	minOptions := int(^uint(0) >> 1) // MaxInt

	for _, c := range candidates {
		if paired[c.ID] || c.ID == player.ID || havePlayed(player, c) {
			continue
		}

		options := countAvailableOpponents(c, candidates, paired)
		if best == nil || options < minOptions {
			best = c
			minOptions = options
		}
	}
	return best
}

// setPairing sets up the round result for both players
func setPairing(p1, p2 *models.Player, roundNum int) {
	roundIdx := roundNum - 1

	for len(p1.Results) <= roundIdx {
		p1.Results = append(p1.Results, models.RoundResult{OpponentID: 0, Win: nil})
	}
	for len(p2.Results) <= roundIdx {
		p2.Results = append(p2.Results, models.RoundResult{OpponentID: 0, Win: nil})
	}

	p1.Results[roundIdx] = models.RoundResult{
		OpponentID: p2.ID,
		Win:        nil,
	}
	p2.Results[roundIdx] = models.RoundResult{
		OpponentID: p1.ID,
		Win:        nil,
	}
}

// setBye gives a player a bye (free win)
func setBye(player *models.Player, roundNum int) {
	roundIdx := roundNum - 1
	for len(player.Results) <= roundIdx {
		player.Results = append(player.Results, models.RoundResult{OpponentID: 0, Win: nil})
	}
	player.Results[roundIdx] = models.RoundResult{
		OpponentID: 0,
		IsBye:      true,
		Win:        ptrBool(true),
	}
}

// havePlayed checks if two players have already played each other in previous rounds
func havePlayed(p1, p2 *models.Player) bool {
	for _, result := range p1.Results {
		if result.OpponentID == p2.ID {
			return true
		}
	}
	return false
}

func ptrBool(b bool) *bool {
	return &b
}
