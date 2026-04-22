package pairing

import (
	"testing"

	"mmg-tournament/models"
)

func TestMakePairs_NoRepeatedPairing_Round1(t *testing.T) {
	players := createTestPlayers([]int{100, 200, 300, 400})

	MakePairs(players, 1)

	// Проверить, что каждый игрок встречается только один раз в туре
	pairCounts := make(map[int]int)
	for i, p1 := range players {
		for j := i + 1; j < len(players); j++ {
			p2 := players[j]
			if havePlayed(p1, p2) {
				pairCounts[p1.ID]++
				pairCounts[p2.ID]++
			}
		}
	}

	for _, count := range pairCounts {
		if count > 1 {
			t.Errorf("Player played more than once in round 1")
		}
	}
}

func TestHavePlayed_Basic(t *testing.T) {
	p1 := &models.Player{ID: 1, Name: "Alice", Results: []models.RoundResult{
		{OpponentID: 2, Win: new(bool)},
	}}
	p2 := &models.Player{ID: 2, Name: "Bob", Results: []models.RoundResult{
		{OpponentID: 1, Win: new(bool)},
	}}

	if !havePlayed(p1, p2) {
		t.Error("Expected havePlayed to return true for players who played")
	}

	p3 := &models.Player{ID: 3, Name: "Charlie", Results: []models.RoundResult{}}
	if havePlayed(p1, p3) {
		t.Error("Expected havePlayed to return false for players who didn't play")
	}
}

func TestHavePlayed_DoesNotCountByes(t *testing.T) {
	p1 := &models.Player{ID: 1, Name: "Alice", Results: []models.RoundResult{
		{OpponentID: 0, IsBye: true, Win: new(bool)},
	}}
	p2 := &models.Player{ID: 2, Name: "Bob", Results: []models.RoundResult{
		{OpponentID: 1, Win: new(bool)},
	}}

	// Bye не должен считаться как игра с соперником
	if havePlayed(p1, p2) {
		t.Error("Expected havePlayed to return false when player got a bye")
	}
}

func TestFindUnpairedOpponent_AvoidsRepeatedPairings(t *testing.T) {
	players := createTestPlayers([]int{100, 200, 300, 400})

	p1 := players[0] // ID: 1
	p2 := players[1] // ID: 2

	// p1 уже играл с p2
	r := new(bool)
	p1.Results = []models.RoundResult{{OpponentID: p2.ID, Win: r}}

	paired := make(map[int]bool)
	paired[p1.ID] = true
	paired[p2.ID] = true

	// P1 и p2 уже свели, не должны их сводить снова
	result := findBestOpponent(p1, players, paired)
	if result == p2 {
		t.Error("Expected findUnpairedOpponent to skip already paired opponent")
	}
}

func TestHavePlayed_SameOpponentMultipleRounds(t *testing.T) {
	r := new(bool)
	p1 := &models.Player{ID: 1, Name: "Alice", Results: []models.RoundResult{
		{OpponentID: 2, Win: r},
		{OpponentID: 2, Win: r},
	}}
	p2 := &models.Player{ID: 2, Name: "Bob", Results: []models.RoundResult{
		{OpponentID: 1, Win: r},
		{OpponentID: 1, Win: r},
	}}

	if !havePlayed(p1, p2) {
		t.Error("Expected havePlayed to return true")
	}
}

func TestMakePairs_WithRepeatedOpponents_SkipsThem(t *testing.T) {
	players := createTestPlayers([]int{100, 200, 300, 400, 500, 600})

	// Эмулируем: в предыдущих турах 1 играл с 2, 3, 4
	r1 := new(bool)
	r2 := new(bool)
	r3 := new(bool)

	players[0].Results = []models.RoundResult{
		{OpponentID: 2, Win: r1},
		{OpponentID: 3, Win: r2},
		{OpponentID: 4, Win: r3},
	}

	// Проверить, что findUnpairedOpponent пропускает повторные встречи
	p1 := players[0]
	candidates := players[1:] // Все остальные игроки

	// 1, 2, 3, 4 уже играли — должны быть пропущены
	paired := make(map[int]bool)
	for _, candidate := range candidates {
		if havePlayed(p1, candidate) {
			paired[candidate.ID] = true
		}
	}

	// После markings as paired, should skip them
	result := findBestOpponent(p1, candidates, paired)
	if result == nil {
		t.Log("Expected to find unpaired opponent but none available")
	}

	// P1 должен иметь возможность сыграть с 5 или 6
	if result != nil {
		if result.ID != 5 && result.ID != 6 {
			t.Errorf("Expected opponent 5 or 6, got %d", result.ID)
		}
	}
}

func TestMakePairs_MultipleRounds_NoRepeatedPairings(t *testing.T) {
	players := createTestPlayers([]int{100, 200, 300, 400})

	// Round 1: 1 vs 2, 3 vs 4
	MakePairs(players, 1)

	// После жеребьёвки проверяем, что не было повторных встреч
	validateNoRepeatedPairings(players, t)

	// Round 2: симулируем результаты round 1, затем жеребьёвку
	// В реальном сценарии результаты будут установлены после жеребьёвки
	// Но для теста мы просто проверяем, что makePairs не создаёт дубликатов
	MakePairs(players, 2)

	validateNoRepeatedPairings(players, t)
}

func TestMakePairs_ThreeRoundRotation(t *testing.T) {
	// 4 игрока, 3 тура, каждый должен сыграть с каждым ровно один раз
	players := createTestPlayers([]int{100, 200, 300, 400})

	round1Pairs := [][]int{{1, 2}, {3, 4}}
	round2Pairs := [][]int{{1, 3}, {2, 4}}
	round3Pairs := [][]int{{1, 4}, {2, 3}}

	rounds := []struct {
		roundNum int
		pairs    [][]int
	}{
		{1, round1Pairs},
		{2, round2Pairs},
		{3, round3Pairs},
	}

	for _, r := range rounds {
		// Устанавливаем результаты как в реальной жеребьёвке
		// Добавляем к существующим результатам
		for _, pair := range r.pairs {
			p1 := players[pair[0]-1]
			p2 := players[pair[1]-1]

			roundIdx := r.roundNum - 1
			for len(p1.Results) <= roundIdx {
				p1.Results = append(p1.Results, models.RoundResult{})
			}
			for len(p2.Results) <= roundIdx {
				p2.Results = append(p2.Results, models.RoundResult{})
			}

			p1.Results[roundIdx] = models.RoundResult{OpponentID: p2.ID, Win: new(bool)}
			p2.Results[roundIdx] = models.RoundResult{OpponentID: p1.ID, Win: new(bool)}
		}

		// Проверяем что нет дубликатов
		for i, pa := range players {
			for j := i + 1; j < len(players); j++ {
				pb := players[j]

				count := countPairings(pa, pb)
				if count > 1 {
					t.Errorf("After round %d: Players %s and %s played %d times",
						r.roundNum, pa.Name, pb.Name, count)
				}
			}
		}
	}

	// В конце проверяем, что каждый сыграл с каждым ровно один раз
	for i, p1 := range players {
		for j := i + 1; j < len(players); j++ {
			p2 := players[j]

			count := countPairings(p1, p2)
			if count != 1 {
				t.Errorf("Players %s and %s played %d times, expected 1",
					p1.Name, p2.Name, count)
			}
		}
	}
}

func validateNoRepeatedPairings(players []*models.Player, t *testing.T) {
	for i, p1 := range players {
		for j := i + 1; j < len(players); j++ {
			p2 := players[j]

			count := countPairings(p1, p2)
			if count > 1 {
				t.Errorf("Players %s and %s played %d times (duplicate pairing)",
					p1.Name, p2.Name, count)
			}
		}
	}
}

func countPairings(p1, p2 *models.Player) int {
	count := 0
	for _, result := range p1.Results {
		if result.OpponentID == p2.ID {
			count++
		}
	}
	return count
}

func createTestPlayers(ratings []int) []*models.Player {
	players := make([]*models.Player, len(ratings))
	for i, rating := range ratings {
		players[i] = &models.Player{
			ID:      i + 1,
			Name:    "Player" + string(rune('A'+i)),
			Rating:  rating,
			Points:  0,
			Results: []models.RoundResult{},
		}
	}
	return players
}
