# ===== Cloud Run 用 Dockerfile =====
# 1段目: Goコードをビルドするためのステージ。
FROM golang:1.23-bookworm AS builder

# アプリケーションの作業ディレクトリ。
WORKDIR /app

# 依存関係だけを先にコピーし、Dockerレイヤーキャッシュを効かせる。
COPY go.mod go.sum* ./
RUN go mod download

# ソースコード全体をコピー。
COPY . .

# Cloud Run上で動かしやすいように、Linux向けの単一バイナリを作る。
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# 2段目: 実行に必要なものだけを入れる軽量ステージ。
FROM gcr.io/distroless/static-debian12

# ビルド済みバイナリをコピー。
COPY --from=builder /app/server /server

# Cloud Runは PORT 環境変数で待受ポートを渡す。
ENV PORT=8080

# コンテナ起動時にGoサーバを実行する。
ENTRYPOINT ["/server"]
