// ============================================================
// ファイル概要: hackathon-backend/internal/auth/jwt.go
// 役割: ログイン後のJWT生成、検証、認証ミドルウェアによるユーザーID注入を担当します。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"hackathon-backend/internal/httpx"
)

// contextKey は、contextに保存する値のキーです。
// stringをそのままキーにすると他パッケージと衝突し得るため、専用型を使います。
// 【詳細コメント】contextKey は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type contextKey string

// 【詳細コメント】userIDKey は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
const userIDKey contextKey = "userID"

// GenerateToken はユーザーIDからJWTを生成します。
// フロントエンドはこのトークンを localStorage に保存し、Authorizationヘッダで送ります。
// 【詳細コメント】GenerateToken は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func GenerateToken(userID int64, secret string) (string, error) {
	claims := jwt.MapClaims{
		// sub は「このトークンが誰を表すか」を意味する標準的なclaimです。
		"sub": fmt.Sprintf("%d", userID),

		// exp は有効期限です。ハッカソンでは7日程度にしておくと扱いやすいです。
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),

		// iat は発行時刻です。
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Middleware は認証が必要なAPIを保護するHTTPミドルウェアです。
// Authorization: Bearer <token> を検証し、成功した場合だけ次のhandlerを実行します。
// 【詳細コメント】Middleware は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func Middleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := UserIDFromRequest(r, secret)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext はミドルウェアがcontextに入れたログインユーザーIDを取り出します。
// 【詳細コメント】UserIDFromContext は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// UserIDFromRequest はHTTPリクエストのAuthorizationヘッダからユーザーIDを取り出します。
// 【詳細コメント】UserIDFromRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func UserIDFromRequest(r *http.Request, secret string) (int64, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return 0, fmt.Errorf("missing authorization header")
	}

	// 【詳細コメント】prefix は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return 0, fmt.Errorf("invalid authorization header")
	}

	tokenString := strings.TrimPrefix(header, prefix)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid claims")
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return 0, err
	}

	// 【詳細コメント】userID は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var userID int64
	if _, err := fmt.Sscanf(sub, "%d", &userID); err != nil {
		return 0, err
	}
	return userID, nil
}
