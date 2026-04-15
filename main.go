package main

import (
	"fmt"
	"os"
	"strconv"

	"mmg-tournament/csv_handler"
	"mmg-tournament/models"
	"mmg-tournament/pairing"
	"mmg-tournament/scoring"
)

func main() {
	// Parse command line arguments
	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Println("Usage: mmg-tournament <base_filename> [total_rounds] [mcmahon_groups]")
		fmt.Println("       mmg-tournament <base_filename> --round <round_num>")
		fmt.Println("       mmg-tournament <base_filename> --final <total_rounds>")
		os.Exit(1)
	}

	baseFilename := args[0]
	var totalRounds int
	var mcmahonGroups int
	var roundNum int
	var isFinal bool

	// Parse optional arguments
	i := 1
	for i < len(args) {
		switch args[i] {
		case "--round":
			if i+1 >= len(args) {
				fmt.Println("Error: --round requires a number argument")
				os.Exit(1)
			}
			roundNum, _ = strconv.Atoi(args[i+1])
			i += 2
		case "--final":
			isFinal = true
			i++
		default:
			// Try to parse as number
			if val, err := strconv.Atoi(args[i]); err == nil {
				// If we haven't set totalRounds yet, this is totalRounds
				if totalRounds == 0 {
					totalRounds = val
				} else if mcmahonGroups == 0 {
					mcmahonGroups = val
				}
			}
			i++
		}
	}

	// Handle final results mode
	if isFinal {
		if totalRounds == 0 {
			fmt.Println("Error: --final requires total_rounds argument")
			fmt.Println("Usage: mmg-tournament <base_filename> --final <total_rounds>")
			os.Exit(1)
		}
		runFinal(baseFilename, totalRounds)
		return
	}

	// Determine if this is the first round or a subsequent round
	isFirstRound := roundNum == 0 || roundNum == 1

	if isFirstRound {
		// First round requires totalRounds and mcmahonGroups
		if totalRounds == 0 || mcmahonGroups == 0 {
			fmt.Println("Error: First round requires total_rounds and mcmahon_groups")
			fmt.Println("Usage: mmg-tournament <base_filename> <total_rounds> <mcmahon_groups>")
			os.Exit(1)
		}
		roundNum = 1
	}

	// Input and output filenames
	inputFile := csv_handler.GetInputFilename(baseFilename, roundNum)
	outputFile := csv_handler.GetOutputFilename(baseFilename, roundNum)

	var players []*models.Player
	var err error

	if isFirstRound {
		// Read initial player list
		fmt.Printf("Reading initial player list from %s...\n", inputFile)
		players, err = csv_handler.ReadInitialPlayers(inputFile)
		if err != nil {
			fmt.Printf("Error reading player list: %v\n", err)
			os.Exit(1)
		}

		if len(players) == 0 {
			fmt.Println("Error: No players found in the input file")
			os.Exit(1)
		}

		fmt.Printf("Found %d players\n", len(players))

		// Sort players by rating
		models.SortPlayersByRating(players)

		// Assign McMahon groups
		scoring.AssignMcMahonGroups(players, mcmahonGroups)

		// Assign unique IDs
		scoring.AssignIDs(players)

		// Calculate initial MMS points
		maxGroup := mcmahonGroups
		for _, p := range players {
			mms := models.CalculateMMS(p.McMahonGroup, maxGroup)
			p.Points = mms // No earned points yet
		}

		// Ensure results slice is big enough
		for _, p := range players {
			p.EnsureResults(totalRounds)
		}

		// Make pairs for the first round
		fmt.Println("Making pairs for round 1...")
		pairing.MakePairs(players, roundNum)

		// Update places
		scoring.UpdatePlaces(players)

	} else {
		// Subsequent round - read tournament file with results
		fmt.Printf("Reading tournament data from %s...\n", inputFile)
		players, totalRounds, err = csv_handler.ReadTournamentFile(inputFile)
		if err != nil {
			fmt.Printf("Error reading tournament file: %v\n", err)
			os.Exit(1)
		}

		if len(players) == 0 {
			fmt.Println("Error: No players found in the input file")
			os.Exit(1)
		}

		fmt.Printf("Found %d players\n", len(players))

		// Recalculate points and coefficients based on previous round results
		fmt.Printf("Calculating coefficients after round %d...\n", roundNum-1)
		scoring.CalculateCoefficients(players, roundNum-1)

		// Sort players according to tournament rules
		models.SortPlayers(players)

		// Make pairs for the current round
		fmt.Printf("Making pairs for round %d...\n", roundNum)
		pairing.MakePairs(players, roundNum)

		// Update places
		scoring.UpdatePlaces(players)
	}

	// Write output file
	fmt.Printf("Writing output to %s...\n", outputFile)
	err = csv_handler.WriteTournamentFile(outputFile, players, totalRounds)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

// runFinal handles the final results calculation mode
func runFinal(baseFilename string, totalRounds int) {
	// Read the last round file (the one with all results)
	inputFile := fmt.Sprintf("%s.%03d.csv", baseFilename, totalRounds)

	fmt.Printf("Reading final tournament data from %s...\n", inputFile)
	players, detectedRounds, err := csv_handler.ReadTournamentFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading tournament file: %v\n", err)
		os.Exit(1)
	}

	if len(players) == 0 {
		fmt.Println("Error: No players found in the input file")
		os.Exit(1)
	}

	fmt.Printf("Found %d players\n", len(players))

	// Verify totalRounds matches
	if detectedRounds != totalRounds {
		fmt.Printf("Warning: detected %d rounds in file, but %d specified. Using %d.\n",
			detectedRounds, totalRounds, totalRounds)
	}

	// Check that all results are entered
	fmt.Printf("Checking that all %d rounds are played for all players...\n", totalRounds)
	if err := scoring.CheckAllResultsPlayed(players, totalRounds); err != nil {
		fmt.Printf("Error: Not all results are entered: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("All results verified!")

	// Calculate final points and coefficients
	fmt.Println("Calculating final points and coefficients...")
	scoring.FinalizeCoefficients(players)

	// Sort players according to final ranking rules
	models.SortPlayers(players)

	// Update places
	scoring.UpdatePlaces(players)

	// Write final results file
	outputFile := fmt.Sprintf("%s.final.csv", baseFilename)
	fmt.Printf("Writing final results to %s...\n", outputFile)
	err = csv_handler.WriteTournamentFile(outputFile, players, totalRounds)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Final Tournament Results ===")
	fmt.Printf("%-4s %-25s %5s %4s %6s %6s %6s\n",
		"Место", "Имя", "Рейтинг", "Очки", "Бергер", "Бухгольц", "Группа")
	fmt.Println("------------------------------------------------------------------------")
	for _, p := range players {
		fmt.Printf("%-4d %-25s %5d %4d %6.1f %6.1f %6d\n",
			p.Place, p.Name, p.Rating, p.Points, p.Berger, p.Buchholz, p.McMahonGroup)
	}

	fmt.Println("\nDone!")
}
