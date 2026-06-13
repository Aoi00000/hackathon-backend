package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client はGemini APIを呼び出すための薄いラッパーです。
// 公式SDKを使う方法もありますが、ハッカソンでは依存を増やしすぎないためRESTで実装しています。
type Client struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// NewClient はGeminiクライアントを初期化します。
func NewClient(apiKey string, model string) *Client {
	return &Client{
		APIKey: apiKey,
		Model:  model,
		HTTP: &http.Client{
			// 外部APIが詰まったときにサーバ全体が待ち続けないよう、タイムアウトを設定します。
			Timeout: 20 * time.Second,
		},
	}
}

// generateContentRequest はGemini generateContent APIに送るJSONです。
// 必要最小限の構造だけを定義しています。
type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

// generateContentResponse はGemini APIのレスポンスからテキストだけを取り出すための構造体です。
type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateText はプロンプトをGeminiに送り、生成されたテキストを返します。
func (c *Client) GenerateText(prompt string) (string, error) {
	// APIキーが未設定・ダミー値のままだと、Gemini APIは必ず失敗します。
	// 画面上で原因を追いやすいよう、外部APIを呼ぶ前に明示的なエラーにします。
	apiKey := strings.TrimSpace(c.APIKey)
	if apiKey == "" || apiKey == "dummy" || strings.Contains(apiKey, "your-gemini") {
		return "", fmt.Errorf("GEMINI_API_KEYが未設定です。Google AI Studioで取得したAPIキーをhackathon-backend/.envに設定し、バックエンドを再起動してください")
	}

	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}

	reqBody := generateContentRequest{
		Contents: []content{
			{
				Parts: []part{
					{Text: prompt},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	responseBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("Gemini APIがHTTP %dを返しました。APIキー、利用可能なモデル名、APIの有効化状態を確認してください。レスポンス: %s", res.StatusCode, string(responseBytes))
	}

	var parsed generateContentResponse
	if err := json.Unmarshal(responseBytes, &parsed); err != nil {
		return "", err
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini api returned empty response")
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// BuildDescriptionPrompt は商品説明生成用のプロンプトを組み立てます。
// AIを単なる飾りにせず、出品者の負担を下げるUXとして使うための機能です。
func BuildDescriptionPrompt(title, category, conditionText, keywords string) string {
	return fmt.Sprintf(`あなたは日本語のフリマアプリの商品説明作成アシスタントです。
以下の商品情報をもとに、購入者が安心して判断できる商品説明を作ってください。

条件:
- 日本語で書く
- 誇張しすぎない
- 状態、用途、注意点が分かる
- 300字以内
- 箇条書きではなく自然な文章にする

商品名: %s
カテゴリ: %s
状態: %s
出品者メモ: %s`, title, category, conditionText, keywords)
}

// BuildItemQAPrompt は商品Q&A用のプロンプトを組み立てます。
// 購入者が商品詳細を読み解きやすくすることを目的にしています。
func BuildItemQAPrompt(title, description, category, conditionText, question string) string {
	return fmt.Sprintf(`あなたはフリマアプリの購入相談アシスタントです。
以下の商品情報だけを根拠に、購入検討者の質問に答えてください。
分からないことは推測で断定せず、「出品者に確認してください」と伝えてください。

商品名: %s
カテゴリ: %s
状態: %s
商品説明: %s

質問: %s`, title, category, conditionText, description, question)
}
