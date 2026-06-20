// ============================================================
// ファイル概要: hackathon-backend/internal/db/db.go
// 役割: MySQL接続プールを作成し、起動時の疎通確認を行います。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
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
// 【詳細コメント】Open は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func Open(cfg config.Config) (*sql.DB, error) {
	// parseTime=true は、MySQLのDATETIMEをGoのtime.Timeとして扱うための設定です。
	// Cloud SQLやMySQLコンテナのDATETIMEはUTC相当で保存される前提に統一します。
	// loc=UTCにしてGo側ではUTCとして受け取り、フロントエンドでAsia/Tokyo表示に変換します。
	// ここをAsia/Tokyoにすると、UTCで保存された12:04を日本時間12:04として解釈してしまい、
	// 実際の21:04より9時間前に表示される原因になります。
	dsnParams := "parseTime=true&charset=utf8mb4&loc=UTC"

	// 【詳細コメント】dsn は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
