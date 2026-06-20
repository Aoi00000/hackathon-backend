// ============================================================
// ファイル概要: hackathon-backend/internal/repository/ai_chat_repository.go
// 役割: AI対話スレッドとメッセージ履歴をDBへ保存・取得・削除する永続化層です。
//
// ============================================================
// 実装詳細メモ:
// AIチャットのスレッドとメッセージ履歴をDBへ保存します。
// ユーザーIDで必ず絞り込むことで、他ユーザーの相談履歴を参照・削除できないようにしています。
// Package repository の ai_chat_repository は、AI対話ページのスレッドと会話履歴を扱います。
//
// 商品コメントやDMとは別に、AIとの一般相談を「話題ごとのスレッド」として保存します。
// これにより、ユーザーは「休日の相談」「勉強環境の相談」「模様替え相談」などを分けて残せます。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"hackathon-backend/internal/models"
)

// AIChatRepository は ai_chat_threads / ai_chat_messages へのDB操作をまとめます。
// Handler からは「スレッドを作る」「履歴を見る」「1往復を保存する」という意味のある単位で呼び出します。
type AIChatRepository struct{ DB *sql.DB }

// normalizeThreadTitle は、空タイトルや長すぎるタイトルをUIで扱いやすい長さに整えます。
// DBのVARCHAR上限より手前で切ることで、入力が長くてもAPIエラーになりにくくします。
func normalizeThreadTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新しい相談"
	}
	runes := []rune(title)
	if len(runes) > 40 {
		title = string(runes[:40]) + "…"
	}
	return title
}

// titleFromMessage は、新規スレッド作成時に最初の質問から自然なスレッド名を作ります。
// ユーザーがタイトル入力を省略しても、サイドバー上で話題を見分けやすくするためです。
func titleFromMessage(message string) string {
	message = strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if message == "" {
		return "新しい相談"
	}
	return normalizeThreadTitle(message)
}

// scanAIChatThread は SELECT結果を models.AIChatThread へ詰める共通処理です。
// 複数のSELECTでScan順を揃え、列追加時の修正漏れを減らします。
func scanAIChatThread(scanner interface{ Scan(dest ...any) error }) (models.AIChatThread, error) {
	var thread models.AIChatThread
	err := scanner.Scan(&thread.ID, &thread.UserID, &thread.Title, &thread.CreatedAt, &thread.UpdatedAt)
	return thread, err
}

// scanAIChatMessage は SELECT結果を models.AIChatMessage へ詰める共通処理です。
// notice はNULL許容なので sql.NullString、used_fallback はMySQLのBOOLEANをintとして受けます。
func scanAIChatMessage(scanner interface{ Scan(dest ...any) error }) (models.AIChatMessage, error) {
	var msg models.AIChatMessage
	var notice sql.NullString
	var usedFallback int
	err := scanner.Scan(&msg.ID, &msg.ThreadID, &msg.Role, &msg.Body, &notice, &usedFallback, &msg.CreatedAt)
	if notice.Valid {
		msg.Notice = notice.String
	}
	msg.UsedFallback = usedFallback == 1
	return msg, err
}

// ListThreads はログインユーザー本人のAI対話スレッドだけを新しい順に返します。
// 他ユーザーの会話履歴を誤って見せないよう、必ず user_id 条件を付けます。
func (r *AIChatRepository) ListThreads(ctx context.Context, userID int64) ([]models.AIChatThread, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, user_id, title, created_at, updated_at FROM ai_chat_threads WHERE user_id = ? ORDER BY updated_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	threads := []models.AIChatThread{}
	for rows.Next() {
		thread, err := scanAIChatThread(rows)
		if err != nil {
			return nil, err
		}
		threads = append(threads, thread)
	}
	return threads, rows.Err()
}

// CreateThread は新しいAI対話スレッドを作成します。
// タイトルだけを先に作ることも、最初の質問送信時に自動作成することもできます。
func (r *AIChatRepository) CreateThread(ctx context.Context, userID int64, title string) (models.AIChatThread, error) {
	title = normalizeThreadTitle(title)
	result, err := r.DB.ExecContext(ctx, `INSERT INTO ai_chat_threads (user_id, title) VALUES (?, ?)`, userID, title)
	if err != nil {
		return models.AIChatThread{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.AIChatThread{}, err
	}
	return r.FindThread(ctx, userID, id)
}

// FindThread は、指定IDのスレッドがログインユーザー本人のものか確認しながら取得します。
// 所有者チェックをこの層へ集約することで、Handler側の分岐を単純にします。
func (r *AIChatRepository) FindThread(ctx context.Context, userID, threadID int64) (models.AIChatThread, error) {
	return scanAIChatThread(r.DB.QueryRowContext(ctx, `SELECT id, user_id, title, created_at, updated_at FROM ai_chat_threads WHERE id = ? AND user_id = ?`, threadID, userID))
}

// DeleteThread は、ユーザー本人のAI対話スレッドを削除します。
// ai_chat_messages は外部キーON DELETE CASCADEでまとめて削除されます。
func (r *AIChatRepository) DeleteThread(ctx context.Context, userID, threadID int64) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM ai_chat_threads WHERE id = ? AND user_id = ?`, threadID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("削除できるAI対話スレッドが見つかりません")
	}
	return nil
}

// ListMessages は、指定スレッドの会話履歴を古い順に返します。
// JOINで thread.user_id を確認するため、他人のthreadIDを指定しても0件になります。
func (r *AIChatRepository) ListMessages(ctx context.Context, userID, threadID int64) ([]models.AIChatMessage, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT m.id, m.thread_id, m.role, m.body, m.notice, m.used_fallback, m.created_at
		FROM ai_chat_messages m
		JOIN ai_chat_threads t ON t.id = m.thread_id
		WHERE t.id = ? AND t.user_id = ?
		ORDER BY m.created_at ASC, m.id ASC`, threadID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []models.AIChatMessage{}
	for rows.Next() {
		msg, err := scanAIChatMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// InsertMessage は、1件のAI対話メッセージを保存し、親スレッドの更新時刻も進めます。
// roleはDBのENUM制約に合わせて user / assistant のみに限定します。
func (r *AIChatRepository) InsertMessage(ctx context.Context, threadID int64, role, body, notice string, usedFallback bool) (models.AIChatMessage, error) {
	role = strings.TrimSpace(role)
	if role != "user" && role != "assistant" {
		return models.AIChatMessage{}, fmt.Errorf("AI対話メッセージのroleが不正です")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return models.AIChatMessage{}, fmt.Errorf("AI対話メッセージ本文が空です")
	}
	usedFallbackInt := 0
	if usedFallback {
		usedFallbackInt = 1
	}
	result, err := r.DB.ExecContext(ctx, `INSERT INTO ai_chat_messages (thread_id, role, body, notice, used_fallback) VALUES (?, ?, ?, NULLIF(?, ''), ?)`, threadID, role, body, strings.TrimSpace(notice), usedFallbackInt)
	if err != nil {
		return models.AIChatMessage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.AIChatMessage{}, err
	}
	if _, err := r.DB.ExecContext(ctx, `UPDATE ai_chat_threads SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, threadID); err != nil {
		return models.AIChatMessage{}, err
	}
	return r.FindMessageByID(ctx, id)
}

// FindMessageByID は、保存直後の1件をAPIレスポンスとして返すための補助関数です。
func (r *AIChatRepository) FindMessageByID(ctx context.Context, messageID int64) (models.AIChatMessage, error) {
	return scanAIChatMessage(r.DB.QueryRowContext(ctx, `SELECT id, thread_id, role, body, notice, used_fallback, created_at FROM ai_chat_messages WHERE id = ?`, messageID))
}

// BuildThreadTitleFromMessage は、Handlerから使うための小さな公開ラッパーです。
// タイトル生成ルールをRepository内に置き、UIとAPIの両方で一貫した表示名にします。
func BuildThreadTitleFromMessage(message string) string { return titleFromMessage(message) }
