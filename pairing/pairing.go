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

		for i := 0; i < len(groupPlayers); i++ {
			if paired[groupPlayers[i].ID] {
				continue
			}

			pairedWith := findUnpairedOpponent(groupPlayers[i], groupPlayers, paired)

			if pairedWith != nil {
				setPairing(groupPlayers[i], pairedWith, roundNum)
				paired[groupPlayers[i].ID] = true
				paired[pairedWith.ID] = true
			} else {
				setBye(groupPlayers[i], roundNum)
				paired[groupPlayers[i].ID] = true
			}
		}
	}
}

// makePairsSubsequent pairs players based on current points for rounds 2+
func makePairsSubsequent(players []*models.Player, roundNum int) {
	paired := make(map[int]bool)

	type pointsGroup struct {
		points  int
		players []*models.Player
	}

	var groups []pointsGroup
	pointsMap := make(map[int][]*models.Player)

	for _, p := range players {
		pointsMap[p.Points] = append(pointsMap[p.Points], p)
	}

	pointsVals := make([]int, 0, len(pointsMap))
	for pts := range pointsMap {
		pointsVals = append(pointsVals, pts)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(pointsVals)))

	for _, pts := range pointsVals {
		groups = append(groups, pointsGroup{points: pts, players: pointsMap[pts]})
	}

	for gi := 0; gi < len(groups); gi++ {
		groupPlayers := groups[gi].players

		for i := 0; i < len(groupPlayers); i++ {
			if paired[groupPlayers[i].ID] {
				continue
			}

			pairedWith := findUnpairedOpponent(groupPlayers[i], groupPlayers, paired)

			if pairedWith == nil {
				for gj := gi + 1; gj < len(groups); gj++ {
					pairedWith = findUnpairedOpponent(groupPlayers[i], groups[gj].players, paired)
					if pairedWith != nil {
						break
					}
				}
			}

			if pairedWith != nil {
				setPairing(groupPlayers[i], pairedWith, roundNum)
				paired[groupPlayers[i].ID] = true
				paired[pairedWith.ID] = true
			} else {
				setBye(groupPlayers[i], roundNum)
				paired[groupPlayers[i].ID] = true
			}
		}
	}
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

// findUnpairedOpponent finds the next unpaired opponent that the player hasn't played yet
func findUnpairedOpponent(player *models.Player, candidates []*models.Player, paired map[int]bool) *models.Player {
	for _, candidate := range candidates {
		if paired[candidate.ID] || candidate.ID == player.ID {
			continue
		}

		if havePlayed(player, candidate) {
			continue
		}

		return candidate
	}

	return nil
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
