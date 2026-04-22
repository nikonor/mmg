package csv_handler

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"mmg-tournament/models"
)

// CSVHandler encapsulates CSV file operations with base filename and round number
type CSVHandler struct {
	baseFilename string
	roundNum     int
}

// New creates a new CSVHandler
func New(baseFilename string, roundNum int) *CSVHandler {
	return &CSVHandler{
		baseFilename: baseFilename,
		roundNum:     roundNum,
	}
}

// GetInputFilename returns the input filename based on base_filename and round_num
func (h *CSVHandler) GetInputFilename() string {
	if h.roundNum <= 1 {
		return h.baseFilename + ".csv"
	}
	// For round 2, we read base_filename.001.csv
	prevRound := h.roundNum - 1
	return fmt.Sprintf("%s.%03d.csv", h.baseFilename, prevRound)
}

// GetOutputFilename returns the output filename for the given round
func (h *CSVHandler) GetOutputFilename() string {
	return fmt.Sprintf("%s.%03d.csv", h.baseFilename, h.roundNum)
}

// ReadInitialPlayers reads the initial player list CSV (name,rating)
func (h *CSVHandler) ReadInitialPlayers() ([]*models.Player, error) {
	filename := h.GetInputFilename()
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var players []*models.Player
	for _, record := range records {
		if len(record) < 2 {
			continue // skip empty lines
		}

		name := strings.TrimSpace(record[0])
		ratingStr := strings.TrimSpace(record[1])
		if name == "" || ratingStr == "" {
			continue
		}

		rating, err := strconv.Atoi(ratingStr)
		if err != nil {
			return nil, fmt.Errorf("invalid rating for %s: %w", name, err)
		}

		players = append(players, models.NewPlayer(name, rating))
	}

	return players, nil
}

// ReadTournamentFile reads a tournament CSV file with results
// Returns players and the detected totalRounds (based on number of result columns)
func (h *CSVHandler) ReadTournamentFile() ([]*models.Player, int, error) {
	filename := h.GetInputFilename()
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read CSV: %w", err)
	}

	var players []*models.Player
	totalRounds := 0

	for _, record := range records {
		if len(record) < 6 {
			continue
		}

		// Detect totalRounds from first record (columns after index 7 are round results)
		if totalRounds == 0 {
			totalRounds = len(record) - 8
			if totalRounds < 1 {
				totalRounds = 1
			}
		}

		id, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid ID: %w", err)
		}

		name := strings.TrimSpace(record[1])
		rating, err := strconv.Atoi(strings.TrimSpace(record[2]))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid rating for %s: %w", name, err)
		}

		group, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid group for %s: %w", name, err)
		}

		points, err := strconv.Atoi(strings.TrimSpace(record[4]))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid points for %s: %w", name, err)
		}

		place, err := strconv.Atoi(strings.TrimSpace(record[5]))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid place for %s: %w", name, err)
		}

		berger := 0.0
		buchholz := 0.0
		if len(record) > 6 {
			berger, _ = strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
		}
		if len(record) > 7 {
			buchholz, _ = strconv.ParseFloat(strings.TrimSpace(record[7]), 64)
		}

		player := &models.Player{
			ID:           id,
			Name:         name,
			Rating:       rating,
			McMahonGroup: group,
			Points:       points,
			Place:        place,
			Berger:       berger,
			Buchholz:     buchholz,
			Results:      make([]models.RoundResult, 0, totalRounds),
		}

		// Parse round results (starting from column 8, index 8)
		for i := 8; i < len(record); i++ {
			result := models.ParseRoundResult(record[i])
			player.Results = append(player.Results, result)
		}

		// Ensure we have enough slots for total rounds
		for len(player.Results) < totalRounds {
			player.Results = append(player.Results, models.RoundResult{OpponentID: 0, Win: nil})
		}

		players = append(players, player)
	}

	// Validate tournament data consistency up to previous round
	validationResult := ValidateTournamentData(players, h.roundNum-1)
	if !validationResult.IsValid {
		return players, totalRounds, fmt.Errorf("tournament data validation failed:%s", validationResult.ErrorString())
	}

	return players, totalRounds, nil
}

// WriteTournamentFile writes the tournament state to a CSV file
func (h *CSVHandler) WriteTournamentFile(players []*models.Player, totalRounds int) error {
	filename := h.GetOutputFilename()
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Update places
	for i, p := range players {
		p.Place = i + 1
	}

	for _, p := range players {
		record := make([]string, 0, 8+totalRounds)
		record = append(record,
			strconv.Itoa(p.ID),
			p.Name,
			strconv.Itoa(p.Rating),
			strconv.Itoa(p.McMahonGroup),
			strconv.Itoa(p.Points),
			strconv.Itoa(p.Place),
			fmt.Sprintf("%.1f", p.Berger),
			fmt.Sprintf("%.1f", p.Buchholz),
		)

		// Add round results
		for i := 0; i < totalRounds; i++ {
			if i < len(p.Results) {
				record = append(record, p.Results[i].String())
			} else {
				record = append(record, "?")
			}
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}
