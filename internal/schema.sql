CREATE TABLE IF NOT EXISTS players (
    id INTEGER PRIMARY KEY,
    discord_id INTEGER UNIQUE,
    tokens INTEGER DEFAULT 100,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
    );

CREATE TABLE IF NOT EXISTS gamemode (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS player_ratings (
    player_id INTEGER,
    GameModeID INTEGER,
    elo INT DEFAULT 1000,
    matches INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (player_id, GameModeID),
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (GameModeID) REFERENCES gamemode(id)
    );

CREATE TABLE IF NOT EXISTS queue (
    discord_id INTEGER UNIQUE,
    GameModeID INTEGER,
    is_matched BOOLEAN DEFAULT FALSE,
    is_unqueued BOOLEAN DEFAULT FALSE,
    timestamp_queued TEXT,
    timestamp_matched TEXT,
    timestamp_unqueued TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (GameModeID) REFERENCES gamemode(id)
    );

CREATE TABLE IF NOT EXISTS matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player1 INTEGER,
    player2 INTEGER,
    GameModeID INTEGER,
    thread_id INTEGER,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (GameModeID) REFERENCES gamemode(id)
    );

CREATE TABLE IF NOT EXISTS match_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player1 INTEGER,
    player2 INTEGER,
    winner INTEGER,
    GameModeID INTEGER,
    elo_before_winner INTEGER,
    elo_after_winner INTEGER,
    elo_before_loser INTEGER,
    elo_after_loser INTEGER,
    datetime TEXT,
    FOREIGN KEY (GameModeID) REFERENCES gamemode(id)
    );

INSERT OR IGNORE INTO gamemode (id, name) VALUES (1, 'land');
INSERT OR IGNORE INTO gamemode (id, name) VALUES (2, 'conquest');
INSERT OR IGNORE INTO gamemode (id, name) VALUES (3, 'domination');
INSERT OR IGNORE INTO gamemode (id, name) VALUES (4, 'luckytest');

CREATE TABLE IF NOT EXISTS bets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER,
    bettor_id INTEGER,
    bet_side INTEGER,
    amount INTEGER,
    placed_at TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (bettor_id) REFERENCES players(id)
    );

CREATE TABLE IF NOT EXISTS user_rewards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    reward_name TEXT,
    role_id INTEGER,
    awarded_at TEXT,
    expires_at TEXT NULL,
    FOREIGN KEY (user_id) REFERENCES players(id) ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT DEFAULT (datetime('now')),
    command TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    user_name TEXT NOT NULL
    );

INSERT INTO user_rewards (user_id, reward_name, role_id, awarded_at, expires_at)
VALUES
    (1, 'Champion', 123456789, datetime('now'), NULL),
    (2, 'Elite Gambler', 987654321, datetime('now'), datetime('now', '+30 days')),
    (3, 'High Roller', 567891234, datetime('now'), datetime('now', '+60 days'));
