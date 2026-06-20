// ============================================================
// ファイル概要: hackathon-backend/internal/ai/gemini.go
// 役割: Gemini / Vertex AI 呼び出しと、デモを止めないためのローカルフォールバック文生成を担当します。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
// Package ai は、Gemini API / Vertex AI / ローカルフォールバックを隠蔽する層です。
//
// 画面側やHandler側がAIプロバイダの違いを意識しないよう、GenerateText だけを公開します。
// Vertex AIの429や一時失敗に対しては短い指数バックオフを行い、失敗時はHandler側でローカル生成へ落とします。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
)

// Client はAI生成を呼び出すための薄いラッパーです。
// AI StudioのAPIキー方式と、研修資料にあるVertex AI方式の両方を扱えるようにしています。
// 【詳細コメント】Client は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type Client struct {
	Provider  string
	APIKey    string
	Model     string
	ProjectID string
	Location  string
	HTTP      *http.Client
}

// 【詳細コメント】NewClient は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func NewClient(provider, apiKey, model, projectID, location string) *Client {
	if provider == "" {
		provider = "ai_studio"
	}
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if location == "" {
		location = "global"
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

// 【詳細コメント】generateContentRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type generateContentRequest struct {
	// 【構造体フィールド】Contents は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Contents []content `json:"contents"`
}

// 【詳細コメント】content は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type content struct {
	// 【構造体フィールド】Parts は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Parts []part `json:"parts"`
}

// 【詳細コメント】part は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type part struct {
	// 【構造体フィールド】Text は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Text string `json:"text"`
}

// 【詳細コメント】generateContentResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				// 【構造体フィールド】Text は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateText は設定されたProviderに応じてGeminiへプロンプトを送ります。
// 【詳細コメント】GenerateText は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (c *Client) GenerateText(prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is empty")
	}

	// 【詳細コメント】lastErr は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			// Vertex AI / Gemini は、モデルやリージョンの共有処理容量が一時的に不足すると
			// 429 ResourceExhausted を返すことがあります。
			// その場合、即座にローカル生成へ落とすよりも、短い指数バックオフで数回だけ
			// 再試行した方が外部AI成功率が上がります。
			delay := time.Duration(500*(1<<(attempt-1))) * time.Millisecond
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			time.Sleep(delay)
		}

		// 【詳細コメント】text は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var text string
		// 【詳細コメント】err は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var err error
		if c.Provider == "vertex" {
			text, err = c.generateTextWithVertex(prompt)
		} else {
			text, err = c.generateTextWithAIStudio(prompt)
		}
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !isRetryableAIError(err) {
			break
		}
	}
	return "", lastErr
}

// 【詳細コメント】isRetryableAIError は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "429") ||
		strings.Contains(text, "resourceexhausted") ||
		strings.Contains(text, "resource exhausted") ||
		strings.Contains(text, "temporarily unavailable") ||
		strings.Contains(text, "deadline exceeded")
}

// generateTextWithAIStudio はGoogle AI StudioのAPIキー方式です。
// 【詳細コメント】generateTextWithAIStudio は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (c *Client) generateTextWithAIStudio(prompt string) (string, error) {
	apiKey := strings.TrimSpace(c.APIKey)
	if apiKey == "" || apiKey == "dummy" || strings.Contains(apiKey, "your-gemini") {
		return "", fmt.Errorf("GEMINI_API_KEYが未設定です。AI_PROVIDER=ai_studioを使う場合は、Google AI Studioで取得した有効なAPIキーをhackathon-backend/.envに設定し、バックエンドを再起動してください")
	}

	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = "gemini-2.5-flash"
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

	// 【詳細コメント】parsed は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
// 【詳細コメント】generateTextWithVertex は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (c *Client) generateTextWithVertex(prompt string) (string, error) {
	projectID := strings.TrimSpace(c.ProjectID)
	if projectID == "" {
		return "", fmt.Errorf("AI_PROVIDER=vertex では GOOGLE_CLOUD_PROJECT または PROJECT_ID が必要です")
	}
	location := strings.TrimSpace(c.Location)
	if location == "" {
		location = "global"
	}
	modelName := strings.TrimSpace(c.Model)
	if modelName == "" {
		modelName = "gemini-2.5-flash"
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

// GenerateTextWithFallback は、外部AIが使える場合は Gemini / Vertex AI の結果を返し、
// 利用枠不足・429・認証未設定・一時障害などで失敗した場合は、
// 画面操作自体を止めないためにローカルの簡易生成文を返します。
//
// このアプリでは「AIで商品説明を生成」「AIに商品について質問する」が
// デモ中に必ず操作できることを重視します。
// そのため、外部AIが落ちた場合でも、エラー画面ではなく
// ルールベースの下書き・回答を返す設計にしています。
// 【詳細コメント】GenerateTextWithFallback は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (c *Client) GenerateTextWithFallback(prompt string, fallback func() string) (string, string, bool, error) {
	text, err := c.GenerateText(prompt)
	if err == nil {
		return text, "", false, nil
	}
	log.Printf("external AI generation failed; falling back to local generator: %v", err)
	fallbackText := strings.TrimSpace(fallback())
	if fallbackText == "" {
		return "", "", false, err
	}
	// 注意文は本文へ混ぜません。
	// 出品画面では商品説明欄にそのまま保存されるため、
	// 本文へ混ぜると「ローカル生成で作成しました」という注意まで商品説明として登録されてしまいます。
	// そこで、本文と注意文を分離してAPIレスポンス側で返します。
	return fallbackText, "※外部AIの利用枠不足または一時的な混雑により、ローカルの簡易生成で作成しました。", true, nil
}

// FallbackDescription は、Gemini / Vertex AI が使えないときの説明文生成です。
// 商品名・カテゴリ・状態・出品者メモだけから、購入者が確認しやすい自然な日本語を作ります。
// 【詳細コメント】FallbackDescription は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func FallbackDescription(title, category, conditionText, keywords string) string {
	title = strings.TrimSpace(title)
	category = strings.TrimSpace(category)
	conditionText = strings.TrimSpace(conditionText)
	keywords = strings.TrimSpace(keywords)
	if keywords == "" {
		keywords = "使用目的や状態を写真とあわせて確認してください"
	}
	return fmt.Sprintf("%sです。カテゴリは%sで、状態は「%s」です。%s。送料込みで出品しています。気になる点があれば、購入前にコメントまたはDMでご確認ください。", title, category, conditionText, keywords)
}

// FallbackItemQA は、Gemini / Vertex AI が使えないときの購入相談回答です。
// 商品情報に書かれている範囲だけを根拠にし、分からない点は出品者確認へ誘導します。
// 【詳細コメント】FallbackItemQA は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func FallbackItemQA(title, description, category, conditionText, question string) string {
	parts := []string{
		fmt.Sprintf("「%s」についての回答です。", strings.TrimSpace(title)),
		fmt.Sprintf("カテゴリは%s、状態は「%s」と登録されています。", strings.TrimSpace(category), strings.TrimSpace(conditionText)),
	}
	if strings.TrimSpace(description) != "" {
		parts = append(parts, "商品説明には次のように記載されています: "+strings.TrimSpace(description))
	}
	if strings.TrimSpace(question) != "" {
		parts = append(parts, "ご質問の内容について、商品説明に明記されていない部分は出品者に確認してください。特にサイズ、付属品、傷や汚れ、動作確認、受け渡し方法は購入前に確認すると安心です。")
	}
	return strings.Join(parts, "\n")
}

// 【詳細コメント】BuildDescriptionPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】BuildItemQAPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】BuildNegotiationPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func BuildNegotiationPrompt(title, description, category, conditionText string, currentPrice, desiredPrice int, role string, commentsSummary string) string {
	return fmt.Sprintf(`あなたはフリマアプリ内の価格交渉アシスタントです。
値下げ交渉では感情的な摩擦が起きやすいため、相手への敬意を保ち、押し付けず、短く丁寧な日本語の文面を作ってください。

条件:
- 現在ユーザーの立場は「%s」
- 価格差が大きい場合は、無理に承諾・要求しない
- 出品者なら「承諾する場合」「難しい場合」「代替案」の3パターンを出す
- 購入検討者なら「丁寧な相談文」「相手が断りやすい余地」「購入意思」の3点を含める
- 250字以内を目安にする
- そのまま公開コメントまたはDMに貼れる文章にする

商品名: %s
カテゴリ: %s
状態: %s
現在価格: %d円
希望金額: %d円
商品説明: %s
公開コメントの要約: %s`, role, title, category, conditionText, currentPrice, desiredPrice, description, commentsSummary)
}

// 【詳細コメント】FallbackNegotiation は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func FallbackNegotiation(title string, currentPrice, desiredPrice int, role string) string {
	// 外部AIが使えない場合でも、値下げ交渉の体験を止めないためのローカル生成です。
	// 商品名・現在価格・希望金額・立場だけから、丁寧で摩擦の少ないテンプレートを作ります。
	diff := currentPrice - desiredPrice
	if role == "seller" {
		if diff <= 0 {
			return fmt.Sprintf("「%s」についてご提案ありがとうございます。ご希望の%d円で対応可能です。購入手続きに進んでいただければ、準備を進めます。よろしくお願いいたします。", strings.TrimSpace(title), desiredPrice)
		}
		if diff <= currentPrice/10 {
			return fmt.Sprintf("「%s」についてご提案ありがとうございます。%d円であれば対応可能です。状態や送料込みである点も踏まえ、この金額でご検討いただけますと幸いです。", strings.TrimSpace(title), desiredPrice)
		}
		counter := currentPrice - currentPrice/20
		return fmt.Sprintf("「%s」についてご提案ありがとうございます。申し訳ありませんが、%d円までのお値下げは現時点では難しいです。送料込みである点もあり、%d円程度であれば検討できます。", strings.TrimSpace(title), desiredPrice, counter)
	}
	return fmt.Sprintf("はじめまして。「%s」の購入を検討しています。大変恐縮ですが、%d円でお譲りいただくことは可能でしょうか。難しい場合は可能な範囲の金額を教えていただけると嬉しいです。よろしくお願いいたします。", strings.TrimSpace(title), desiredPrice)
}

// 【詳細コメント】BuildRecommendationPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func BuildRecommendationPrompt(userName string, itemsSummary string) string {
	return fmt.Sprintf(`あなたはフリマアプリの推薦アシスタントです。
ユーザー名: %s
候補商品:
%s

上記の商品群について、購入検討の観点からおすすめ理由を120字以内で日本語でまとめてください。`, userName, itemsSummary)
}

// 【詳細コメント】BuildItemAnalysisPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func BuildItemAnalysisPrompt(title, description, category, conditionText string, priceYen int, priceInsight string, categoryHints string) string {
	return fmt.Sprintf(`あなたはフリマアプリの購入前チェックを行うAIアシスタントです。
以下の商品情報を読み、購入者の不安を減らすために、次の3項目を日本語で簡潔に出してください。

1. 不安点: 最大3件
2. 購入者が出品者に質問すべきこと: 最大3件
3. 出品文・カテゴリ・状態などの不整合疑い: 最大3件。なければ「大きな不整合は見当たりません」とする

商品名: %s
カテゴリ: %s
状態: %s
価格: %d円
商品説明: %s
価格比較メモ: %s
カテゴリ別レビュー知識: %s

出力形式:
不安点:
- ...
質問候補:
- ...
不整合:
- ...`, title, category, conditionText, priceYen, description, priceInsight, categoryHints)
}

// 【詳細コメント】BuildGeneralChatPrompt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func BuildGeneralChatPrompt(message string) string {
	return fmt.Sprintf(`あなたは大学生向けフリマアプリ内の対話型AIです。
ユーザーの相談に対して、一般的なGeminiのように自然で役立つ日本語で答えてください。
フリマアプリ内の機能なので、回答の最後には、その相談に役立ちそうなおすすめグッズを一般名で3〜6個提示してください。
商品の個別在庫を断定せず、「探してみるとよいもの」として一般名で出してください。

条件:
- 300字以内を目安にする
- 必要なら箇条書きを使う
- 危険・違法・医療断定などは避け、安全な範囲の提案にする
- 最後に必ず「おすすめグッズ:」を付ける

ユーザーの相談:
%s`, message)
}

// 【詳細コメント】FallbackGeneralChat は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func FallbackGeneralChat(message string) string {
	text := strings.ToLower(strings.TrimSpace(message))
	// 【詳細コメント】answer は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var answer string
	// 【詳細コメント】goods は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var goods []string
	switch {
	case strings.Contains(text, "休日") || strings.Contains(text, "遊び") || strings.Contains(text, "出かけ") || strings.Contains(text, "海") || strings.Contains(text, "山"):
		answer = "気分転換をしたいなら、近場の公園散歩、海沿いの散策、山や川のある場所への小旅行などがおすすめです。予定を詰め込みすぎず、写真を撮る時間やカフェで休む時間を入れると、非日常感が出やすくなります。"
		goods = []string{"日焼け止め", "歩きやすいサンダル", "リュック", "モバイルバッテリー", "折りたたみ傘"}
	case strings.Contains(text, "模様替え") || strings.Contains(text, "部屋") || strings.Contains(text, "インテリア") || strings.Contains(text, "家具"):
		answer = "大きな家具を変えなくても、カーテン、ベッドシーツ、照明、小物の色をそろえるだけで部屋の印象はかなり変わります。落ち着いた雰囲気なら青・水色・グレー、温かい雰囲気ならベージュ・木目・淡いオレンジが使いやすいです。"
		goods = []string{"カーテン", "ベッドシーツ", "間接照明", "収納ボックス", "観葉植物"}
	case strings.Contains(text, "勉強") || strings.Contains(text, "試験") || strings.Contains(text, "レポート") || strings.Contains(text, "集中"):
		answer = "集中したいときは、作業時間を25〜50分に区切り、机の上から関係ないものを減らすのが効果的です。最初は難しい教材ではなく、今日終わらせる範囲が見えるタスクから始めると入りやすいです。"
		goods = []string{"タイマー", "ノート", "参考書", "ブックスタンド", "ノイズキャンセリングイヤホン"}
	case strings.Contains(text, "料理") || strings.Contains(text, "自炊") || strings.Contains(text, "キッチン"):
		answer = "自炊を続けたいなら、最初は切る・焼く・保存するの手間を減らす道具をそろえるのがおすすめです。作り置きしやすいメニューから始めると、食費も時間も管理しやすくなります。"
		goods = []string{"保存容器", "フライパン", "包丁", "まな板", "調味料ラック"}
	default:
		answer = "やりたいことを少し具体化して、準備・移動・片付けの手間が小さい形から試すのがおすすめです。まずは今日すぐできる小さな行動に分けると、気軽に始めやすくなります。"
		goods = []string{"ノート", "トートバッグ", "モバイルバッテリー", "収納ケース", "小型ライト"}
	}
	return answer + "\n\nおすすめグッズ:\n- " + strings.Join(goods, "\n- ")
}
