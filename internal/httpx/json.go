// ============================================================
// ファイル概要: hackathon-backend/internal/httpx/json.go
// 役割: JSONレスポンス、JSONリクエスト読み取り、エラーレスポンスを共通化します。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
package httpx

import (
	"encoding/json"
	"net/http"

	"hackathon-backend/internal/models"
)

// WriteJSON は任意の値をJSONレスポンスとして返す補助関数です。
// 各ハンドラに同じ処理を書かないために共通化しています。
// 【詳細コメント】WriteJSON は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError はエラーを {"error":"..."} という形で返す補助関数です。
// フロントエンド側が一貫した形でエラー表示できるようにします。
// 【詳細コメント】WriteError は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, models.ErrorResponse{Error: message})
}

// DecodeJSON はリクエストボディのJSONを構造体に読み込む補助関数です。
// 【詳細コメント】DecodeJSON は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
