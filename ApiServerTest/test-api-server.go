package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBody はリクエストボディの構造体を定義します
type RequestBody struct {
	Message string `json:"message"`
	Data    int    `json:"data"`
}

func main() {
	r := gin.Default()

	// POST /post エンドポイントのハンドラ
	r.POST("/post", func(c *gin.Context) {
		var requestBody RequestBody

		// リクエストボディを構造体にバインド
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// レスポンスボディを作成
		response := gin.H{
			"received_message": requestBody.Message,
			"received_data":    requestBody.Data,
			"status":           "ok",
		}

		// JSON 形式でレスポンスを返す
		c.JSON(http.StatusOK, response)
	})

	// サーバーを起動
	r.Run(":8080") // ポート番号は適宜変更
}
