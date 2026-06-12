package httpx

import (
	"encoding/json"
	"net/http"

	"hackathon-backend/internal/models"
)

// WriteJSON は任意の値をJSONレスポンスとして返す補助関数です。
// 各ハンドラに同じ処理を書かないために共通化しています。
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError はエラーを {"error":"..."} という形で返す補助関数です。
// フロントエンド側が一貫した形でエラー表示できるようにします。
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, models.ErrorResponse{Error: message})
}

// DecodeJSON はリクエストボディのJSONを構造体に読み込む補助関数です。
func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
