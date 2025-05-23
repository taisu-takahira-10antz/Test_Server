set -eux
# e: エラー時にスクリプトを終了する
# u: 未定義の変数を参照したときにエラーにする
# x: 実行コマンドのログを標準エラー出力に出力する

docker start takahira-test

# インスタンスの作成
gcloud spanner instances create takahira-test-instance \
    --config=emulator-config \
    --description="Takahira Test Instance" \
    --nodes=1

# データベースの作成
gcloud spanner databases create takahira-test-databases --instance=takahira-test-instance

# データベースに初期データ追加
spanner-cli -p memory-dev-3dc1b -i takahira-test-instance -d takahira-test-databases -f init-data.sql

# 終了メッセージ
echo "takahira-test SetUp Completed"

# データベースに接続
# spanner-cli -p memory-dev-3dc1b -i takahira-test-instance -d takahira-test-databases