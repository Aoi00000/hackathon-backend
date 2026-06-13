package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"hackathon-backend/internal/config"
)

// Open はMySQLへの接続を作る関数です。
// ローカル開発では TCP 接続、Cloud Run + Cloud SQL では Unix socket 接続を使えるようにしています。
func Open(cfg config.Config) (*sql.DB, error) {
	// parseTime=true は、MySQLのDATETIMEをGoのtime.Timeとして扱うための設定です。
	// loc=Local は、ローカルタイムゾーンとして解釈するための設定です。
	dsnParams := "parseTime=true&charset=utf8mb4&loc=Asia%2FTokyo"

	var dsn string

	// Cloud SQL接続をCloud Runに設定すると、/cloudsql/PROJECT:REGION:INSTANCE のUnix socketが使えます。
	// 資料ではCloud Run側の「Cloud SQL 接続」設定を行う流れが示されているため、それに合わせています。
	if strings.HasPrefix(cfg.MySQLHost, "/cloudsql/") {
		dsn = fmt.Sprintf(
			"%s:%s@unix(%s)/%s?%s",
			cfg.MySQLUser,
			cfg.MySQLPassword,
			cfg.MySQLHost,
			cfg.MySQLDatabase,
			dsnParams,
		)
	} else {
		// ローカル開発では 127.0.0.1:3306 のようなTCPアドレスで接続します。
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s)/%s?%s",
			cfg.MySQLUser,
			cfg.MySQLPassword,
			cfg.MySQLHost,
			cfg.MySQLDatabase,
			dsnParams,
		)
	}

	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Cloud Runではインスタンス数が増えるとDB接続も増えやすいため、接続数を抑えます。
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)

	// Pingで実際に接続できるか確認します。
	if err := database.Ping(); err != nil {
		return nil, err
	}

	return database, nil
}
