package models

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// RoundResult represents the result of a single round for a player
type RoundResult struct {
	OpponentID int   // ID of the opponent (0 for bye)
	IsBye      bool  // true if player got a bye (X)
	Win        *bool // nil if not played yet, true if win, false if loss
}

// Player represents a tournament participant
type Player struct {
	ID           int
	Name         string
	Rating       int
	McMahonGroup int           // Group number (1 = strongest)
	Points       int           // Current total points (MMS + earned)
	Place        int           // Current ranking position
	Berger       float64       // Berger coefficient
	Buchholz     float64       // Buchholz coefficient
	Results      []RoundResult // Results for each round (index 0 = round 1)
}

// String formats player result for CSV column
func (r RoundResult) String() string {
	if r.IsBye {
		return "X"
	}
	if r.OpponentID == 0 {
		return "?"
	}
	if r.Win == nil {
		return fmt.Sprintf("%d:?", r.OpponentID)
	}
	if *r.Win {
		return fmt.Sprintf("%d:1", r.OpponentID)
	}
	return fmt.Sprintf("%d:0", r.OpponentID)
}

// ParseRoundResult parses a round result string from CSV
func ParseRoundResult(s string) RoundResult {
	s = strings.TrimSpace(s)
	if s == "" || s == "?" {
		return RoundResult{OpponentID: 0, Win: nil}
	}
	if s == "X" {
		return RoundResult{OpponentID: 0, IsBye: true, Win: new(true)}
	}

	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return RoundResult{OpponentID: 0, Win: nil}
	}

	oppID, err := strconv.Atoi(parts[0])
	if err != nil {
		return RoundResult{OpponentID: 0, Win: nil}
	}

	result := RoundResult{OpponentID: oppID}
	switch parts[1] {
	case "1":
		result.Win = new(true)
	case "0":
		result.Win = new(false)
	default:
		result.Win = nil
	}

	return result
}

// HasPlayed checks if the round has been played
func (r RoundResult) HasPlayed() bool {
	return r.Win != nil
}

// IsWin returns true if this result is a win
func (r RoundResult) IsWin() bool {
	return r.Win != nil && *r.Win
}

// NewPlayer creates a new player from initial CSV data (name,rating)
func NewPlayer(name string, rating int) *Player {
	return &Player{
		Name:   name,
		Rating: rating,
	}
}

// AssignID assigns a unique ID to the player
func (p *Player) AssignID(id int) {
	p.ID = id
}

// EnsureResults ensures Results slice has enough capacity for totalRounds
func (p *Player) EnsureResults(totalRounds int) {
	for len(p.Results) < totalRounds {
		p.Results = append(p.Results, RoundResult{OpponentID: 0, Win: nil})
	}
}

// GetPointsEarned returns the number of points earned from played games
func (p *Player) GetPointsEarned() int {
	earned := 0
	for _, r := range p.Results {
		if r.IsBye || r.IsWin() {
			earned++
		}
	}
	return earned
}

// GetTotalPoints returns MMS + earned points
func (p *Player) GetTotalPoints() int {
	return p.GetPointsEarned()
}

// CalculateMMS calculates MMS based on group number and total groups
func CalculateMMS(group, totalGroups int) int {
	return totalGroups - group
}

// SortPlayers sorts players according to tournament rules
func SortPlayers(players []*Player) {
	sort.SliceStable(players, func(i, j int) bool {
		pi := players[i]
		pj := players[j]

		// Sort by: Points (desc), Berger (desc), Buchholz (desc), Rating (desc)
		if pi.Points != pj.Points {
			return pi.Points > pj.Points
		}
		if pi.Berger != pj.Berger {
			return pi.Berger > pj.Berger
		}
		if pi.Buchholz != pj.Buchholz {
			return pi.Buchholz > pj.Buchholz
		}
		return pi.Rating > pj.Rating
	})
}

// SortPlayersByRating sorts players by rating descending
func SortPlayersByRating(players []*Player) {
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Rating > players[j].Rating
	})
}
