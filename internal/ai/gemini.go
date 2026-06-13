package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
)

// Client はAI生成を呼び出すための薄いラッパーです。
// AI StudioのAPIキー方式と、研修資料にあるVertex AI方式の両方を扱えるようにしています。
type Client struct {
	Provider  string
	APIKey    string
	Model     string
	ProjectID string
	Location  string
	HTTP      *http.Client
}

func NewClient(provider, apiKey, model, projectID, location string) *Client {
	if provider == "" {
		provider = "ai_studio"
	}
	if model == "" {
		model = "gemini-1.5-flash-002"
	}
	if location == "" {
		location = "asia-northeast1"
	}
	return &Client{
		Provider:  provider,
		APIKey:    apiKey,
		Model:     model,
		ProjectID: projectID,
		Location:  location,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateText は設定されたProviderに応じてGeminiへプロンプトを送ります。
func (c *Client) GenerateText(prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	if c.Provider == "vertex" {
		return c.generateTextWithVertex(prompt)
	}
	return c.generateTextWithAIStudio(prompt)
}

// generateTextWithAIStudio はGoogle AI StudioのAPIキー方式です。
func (c *Client) generateTextWithAIStudio(prompt string) (string, error) {
	apiKey := strings.TrimSpace(c.APIKey)
	if apiKey == "" || apiKey == "dummy" || strings.Contains(apiKey, "your-gemini") {
		return "", fmt.Errorf("GEMINI_API_KEYが未設定です。AI_PROVIDER=ai_studioを使う場合は、Google AI Studioで取得した有効なAPIキーをhackathon-backend/.envに設定し、バックエンドを再起動してください")
	}

	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = "gemini-1.5-flash-002"
	}

	bodyBytes, err := json.Marshal(generateContentRequest{Contents: []content{{Parts: []part{{Text: prompt}}}}})
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
		if res.StatusCode == http.StatusTooManyRequests {
			return "", fmt.Errorf("Gemini APIがHTTP 429を返しました。表示されている RESOURCE_EXHAUSTED / prepayment credits depleted は、AI Studio側の利用枠またはプリペイド残高が尽きたことを示します。研修資料のVertex AI方式へ切り替える場合は、AI_PROVIDER=vertex、GOOGLE_CLOUD_PROJECT、VERTEX_LOCATIONを設定し、gcloud auth application-default loginを実行してください。レスポンス: %s", string(responseBytes))
		}
		return "", fmt.Errorf("Gemini APIがHTTP %dを返しました。APIキー、モデル名、API有効化状態を確認してください。レスポンス: %s", res.StatusCode, string(responseBytes))
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

// generateTextWithVertex は研修資料「独自データを使った生成AIの利用(Go)」に沿ったVertex AI方式です。
// ローカルでは gcloud auth application-default login、本番Cloud Runではサービスアカウント権限が必要です。
func (c *Client) generateTextWithVertex(prompt string) (string, error) {
	projectID := strings.TrimSpace(c.ProjectID)
	if projectID == "" {
		return "", fmt.Errorf("AI_PROVIDER=vertex では GOOGLE_CLOUD_PROJECT または PROJECT_ID が必要です")
	}
	location := strings.TrimSpace(c.Location)
	if location == "" {
		location = "asia-northeast1"
	}
	modelName := strings.TrimSpace(c.Model)
	if modelName == "" {
		modelName = "gemini-1.5-flash-002"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return "", fmt.Errorf("Vertex AIクライアント作成に失敗しました。ローカルでは gcloud auth application-default login を実行し、Vertex AI APIを有効化してください: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("Vertex AIでの生成に失敗しました。プロジェクトID、ロケーション、モデル名、ADC認証、Vertex AI APIの有効化を確認してください: %w", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("vertex ai returned empty response")
	}

	parts := make([]string, 0, len(resp.Candidates[0].Content.Parts))
	for _, p := range resp.Candidates[0].Content.Parts {
		parts = append(parts, fmt.Sprint(p))
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

func BuildDescriptionPrompt(title, category, conditionText, keywords string) string {
	return fmt.Sprintf(`あなたは日本語のフリマアプリの商品説明作成アシスタントです。
以下の商品情報をもとに、購入者が安心して判断できる商品説明を作ってください。

条件:
- 日本語で書く
- 誇張しすぎない
- 状態、用途、注意点が分かる
- 300字以内
- 箇条書きではなく自然な文章にする
- 送料は無料であることを自然に含める

商品名: %s
カテゴリ: %s
状態: %s
出品者メモ: %s`, title, category, conditionText, keywords)
}

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

func BuildRecommendationPrompt(userName string, itemsSummary string) string {
	return fmt.Sprintf(`あなたはフリマアプリの推薦アシスタントです。
ユーザー名: %s
候補商品:
%s

上記の商品群について、購入検討の観点からおすすめ理由を120字以内で日本語でまとめてください。`, userName, itemsSummary)
}
