package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type MatchDB struct {
	DB *sql.DB
}

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		log.Fatal(err)
	}

	schemaSQL, err := os.ReadFile("./internal/schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		log.Fatalf("Failed to execute schema SQL: %v", err)
	}

	log.Println("Database schema created and initialized.")
	return db
}

func NewMatchDB(db *sql.DB) *MatchDB {
	return &MatchDB{DB: db}
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func CreateTables() {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schemaSQL, err := os.ReadFile("./internal/schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		log.Fatalf("Failed to execute schema SQL: %v", err)
	}

	log.Println("Database schema created and initialized.")
}

func (m *MatchDB) LogEvent(command string, userID int64, userName string) {
	_, err := m.DB.Exec(`
		INSERT INTO logs (timestamp, command, user_id, user_name) 
		VALUES (?, ?, ?, ?)`,
		nowRFC3339(), command, userID, userName,
	)
	if err != nil {
		log.Printf("log_event error: %v", err)
	}
}

func (m *MatchDB) GetPlayerID(discordID int64) (int64, error) {
	var id int64
	err := m.DB.QueryRow("SELECT id FROM players WHERE discord_id = ?", discordID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *MatchDB) FetchPlayerRating(discordID int64, gameModeID int) (int, error) {
	var elo int
	query := `
		SELECT elo FROM player_ratings 
		WHERE player_id = (SELECT id FROM players WHERE discord_id = ?) 
		  AND GameModeID = ?
	`
	err := m.DB.QueryRow(query, discordID, gameModeID).Scan(&elo)
	if err != nil {
		if err == sql.ErrNoRows {
			return 1000, nil
		}
		return 0, err
	}
	return elo, nil
}

func (m *MatchDB) AddPlayer(userID int64) {
	now := nowRFC3339()
	stmt := `
		INSERT INTO players (discord_id, created_at, updated_at)
		    VALUES (?, ?, ?)
		    ON CONFLICT(discord_id) DO UPDATE SET updated_at = ?
	`
	_, err := m.DB.Exec(stmt, userID, now, now, now)
	if err != nil {
		log.Printf("Error adding player: %v", err)
	}
}

func (m *MatchDB) AddPlayerMode(discordID int64, gameModeID int) {
	var playerID int64

	err := m.DB.QueryRow("SELECT id FROM players WHERE discord_id = ?", discordID).Scan(&playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Player with Discord ID %d not found", discordID)
			return
		}
		log.Printf("Error fetching player: %v", err)
		return
	}

	stmt := `
		INSERT OR IGNORE INTO player_ratings (player_id, GameModeID, elo, matches, wins)
		VALUES (?, ?, 1000, 0, 0)
	`
	_, err = m.DB.Exec(stmt, playerID, gameModeID)
	if err != nil {
		log.Printf("Error inserting player rating: %v", err)
		return
	}
}

func (m *MatchDB) UpdateElo(discordID int64, gameModeID int, newElo int) {
	stmt := `
		UPDATE player_ratings 
		SET elo = ?
		WHERE player_id = (SELECT id FROM players WHERE discord_id = ?) AND GameModeID = ?
	`

	_, err := m.DB.Exec(stmt, newElo, discordID, gameModeID)
	if err != nil {
		log.Printf("Error updating elo: %v", err)
	}
}

func (m *MatchDB) GetQueuePlayers(gameModeID int) []int64 {
	stmt := `
		SELECT discord_id FROM queue 
		WHERE GameModeID = ? AND is_matched = FALSE AND is_unqueued = FALSE
	`

	rows, err := m.DB.Query(stmt, gameModeID)
	if err != nil {
		log.Printf("Error fetching queue players: %v", err)
		return nil
	}
	defer rows.Close()

	var players []int64
	for rows.Next() {
		var discordID int64
		if err := rows.Scan(&discordID); err != nil {
			log.Printf("Error scanning discord_id: %v", err)
			continue
		}
		players = append(players, discordID)
	}

	return players
}

func (m *MatchDB) GetQueuePlayersCount(gameModeID int) int {
	stmt := `
		SELECT COUNT(*) FROM queue 
		WHERE GameModeID = ? AND is_matched = FALSE AND is_unqueued = FALSE
	`

	var count int
	err := m.DB.QueryRow(stmt, gameModeID).Scan(&count)
	if err != nil {
		log.Printf("Error counting queue players: %v", err)
		return 0
	}

	return count
}

func (m *MatchDB) AddToQueue(discordID int64, gameModeID int) {
	now := nowRFC3339()

	const stmt = `
		INSERT INTO queue (discord_id, GameModeID, timestamp_queued, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(discord_id) DO UPDATE SET
			GameModeID = excluded.GameModeID,
			is_matched = FALSE,
			is_unqueued = FALSE,
			timestamp_queued = excluded.timestamp_queued,
			timestamp_matched = NULL,
			timestamp_unqueued = NULL,
			updated_at = excluded.updated_at
	`

	_, err := m.DB.Exec(stmt, discordID, gameModeID, now, now, now)
	if err != nil {
		log.Printf("[AddToQueue] Exec error: %v", err)
		return
	}

	m.LogEvent("add_to_queue", discordID, fmt.Sprintf("User %d added to queue for mode %d", discordID, gameModeID))
}

func (m *MatchDB) MarkAsMatched(discordID int64) {
	now := nowRFC3339()

	const stmt = `
		UPDATE queue
		SET is_matched = TRUE,
			timestamp_matched = ?,
			updated_at = ?
		WHERE discord_id = ?
	`

	_, err := m.DB.Exec(stmt, now, now, discordID)
	if err != nil {
		log.Printf("[MarkAsMatched] Exec error: %v", err)
		return
	}

	m.LogEvent("mark_as_matched", discordID, fmt.Sprintf("User %d marked as matched", discordID))
}

func (m *MatchDB) MarkAsUnqueued(discordID int64) {
	now := nowRFC3339()

	const stmt = `
		UPDATE queue
		SET is_unqueued = TRUE,
			timestamp_unqueued = ?,
			updated_at = ?
		WHERE discord_id = ?
	`

	_, err := m.DB.Exec(stmt, now, now, discordID)
	if err != nil {
		log.Printf("[MarkAsUnqueued] Exec error: %v", err)
		return
	}

	m.LogEvent("mark_as_unqueued", discordID, fmt.Sprintf("User %d marked as unqueued", discordID))
}

func (m *MatchDB) CreateMatch(player1DiscordID, player2DiscordID int64, gameModeID int, threadID int64) error {
	now := nowRFC3339()

	player1ID, err := m.GetPlayerID(player1DiscordID)
	if err != nil {
		return err
	}
	player2ID, err := m.GetPlayerID(player2DiscordID)
	if err != nil {
		return err
	}

	_, err = m.DB.Exec(`
		INSERT INTO matches (player1, player2, GameModeID, thread_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		player1ID, player2ID, gameModeID, threadID, now, now,
	)
	if err != nil {
		return err
	}

	m.LogEvent("create_match", player1DiscordID, "UnknownUser")
	return nil
}

func (m *MatchDB) RemoveMatch(player1ID, player2ID int64) {
	const stmt = `
		DELETE FROM matches 
		WHERE (player1 = ? AND player2 = ?) 
		   OR (player1 = ? AND player2 = ?)
	`

	res, err := m.DB.Exec(stmt, player1ID, player2ID, player2ID, player1ID)
	if err != nil {
		log.Printf("[RemoveMatch] Exec error: %v", err)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		log.Printf("[RemoveMatch] No match found for players %d and %d", player1ID, player2ID)
		return
	}

	m.LogEvent("remove_match", player1ID, fmt.Sprintf("Removed match between players %d and %d", player1ID, player2ID))
}

func (m *MatchDB) RecordMatchResult(winnerDiscordID, loserDiscordID int64, gameModeID int,
	eloBeforeWinner, eloAfterWinner, eloBeforeLoser, eloAfterLoser int) error {

	now := nowRFC3339()

	winnerID, err := m.GetPlayerID(winnerDiscordID)
	if err != nil {
		return err
	}
	loserID, err := m.GetPlayerID(loserDiscordID)
	if err != nil {
		return err
	}

	_, err = m.DB.Exec(`
		INSERT INTO match_history (player1, player2, winner, GameModeID,
								   elo_before_winner, elo_after_winner,
								   elo_before_loser, elo_after_loser, datetime)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		winnerID, loserID, winnerID, gameModeID,
		eloBeforeWinner, eloAfterWinner,
		eloBeforeLoser, eloAfterLoser,
		now,
	)
	if err != nil {
		return err
	}

	_, err = m.DB.Exec(`
		UPDATE player_ratings
		SET elo = ?, matches = matches + 1, wins = wins + 1
		WHERE player_id = ? AND GameModeID = ?`,
		eloAfterWinner, winnerID, gameModeID,
	)
	if err != nil {
		return err
	}

	_, err = m.DB.Exec(`
		UPDATE player_ratings
		SET elo = ?, matches = matches + 1
		WHERE player_id = ? AND GameModeID = ?`,
		eloAfterLoser, loserID, gameModeID,
	)
	return err
}

func (m *MatchDB) GetQueueStatus(discordID int64) *int {
	var gameModeID int
	err := m.DB.QueryRow(`
		SELECT GameModeID FROM queue
		WHERE discord_id = ? AND is_matched = FALSE AND is_unqueued = FALSE
	`, discordID).Scan(&gameModeID)

	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.Printf("GetQueueStatus error: %v", err)
		return nil
	}
	return &gameModeID
}

func (m *MatchDB) GetMatchDetails(discordID int64) (opponentID *int64, gameModeID *int) {
	var player1, player2 int64
	var gmID int
	err := m.DB.QueryRow(`
		SELECT player1, player2, GameModeID FROM matches
		WHERE player1 = ? OR player2 = ?
	`, discordID, discordID).Scan(&player1, &player2, &gmID)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Printf("GetMatchDetails error: %v", err)
		return nil, nil
	}

	var opponent int64
	if player1 == discordID {
		opponent = player2
	} else {
		opponent = player1
	}

	return &opponent, &gmID
}

func (m *MatchDB) GetPlayerRating(discordID int64, gameModeID int) string {
	var elo int
	err := m.DB.QueryRow(`
		SELECT elo FROM player_ratings
		WHERE player_id = (SELECT id FROM players WHERE discord_id = ?) AND GameModeID = ?
	`, discordID, gameModeID).Scan(&elo)

	if err == sql.ErrNoRows {
		return "N/A"
	} else if err != nil {
		log.Printf("GetPlayerRating error: %v", err)
		return "N/A"
	}
	return fmt.Sprintf("%d", elo)
}

func (m *MatchDB) UpdatePlayerRating(playerID int64, gameModeID, rating int) {
	_, err := m.DB.Exec(`
		UPDATE player_ratings
		SET elo = ?, matches = matches + 1
		WHERE player_id = ? AND GameModeID = ?
	`, rating, playerID, gameModeID)

	if err != nil {
		log.Printf("UpdatePlayerRating error: %v", err)
	}
}

func (m *MatchDB) GetMatchThread(player1ID, player2ID int64, gameModeID int) string {
	var threadID int64
	err := m.DB.QueryRow(`
        SELECT thread_id FROM matches
        WHERE ((player1 = ? AND player2 = ?) OR (player1 = ? AND player2 = ?))
          AND GameModeID = ?`,
		player1ID, player2ID, player2ID, player1ID, gameModeID,
	).Scan(&threadID)

	if err != nil {
		if err == sql.ErrNoRows {
			return ""
		}
		log.Printf("GetMatchThread error: %v", err)
		return ""
	}
	return strconv.FormatInt(threadID, 10)
}
