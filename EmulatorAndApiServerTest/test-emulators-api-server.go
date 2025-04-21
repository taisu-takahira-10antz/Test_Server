package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/gin-gonic/gin"
)

// User はテーブルのレコードを表す構造体
type User struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Money float64 `json:"money"`
}

// spannerClient は Spanner クライアント
var spannerClient *spanner.Client

// dbName は Spanner データベース名
var dbName string

func main() {
	// 環境変数から Spanner エミュレータのホストを取得
	emulatorHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorHost != "" {
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost)
	}

	ctx := context.Background()

	// Spanner クライアントの初期化
	projectID := "memory-dev-3dc1b"                                                                                 // プロジェクトID
	instanceID := "takahira-test-instance"                                                                          // インスタンスID
	dbName = fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, "takahira-test-databases") // データベース名

	client, err := spanner.NewClient(ctx, dbName)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	spannerClient = client
	defer spannerClient.Close()

	r := gin.Default()

	// API エンドポイント
	r.POST("/users/create", createUser)
	r.GET("/users/get/id", getUserId)
	r.GET("/users/get/all", getUserAll)
	r.PUT("/users/update", updateUser)
	r.DELETE("/users/delete", deleteUser)

	r.Run(":8080")
}

// createUser は新しいユーザーを作成する
func createUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Println("Received user data:", user) // ログ出力

	_, err := spannerClient.ReadWriteTransaction(context.Background(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: "INSERT INTO Users (id, name, money) VALUES (@id, @name, @money)",
			Params: map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name,
				"money": user.Money,
			},
		}
		_, err := txn.Update(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "data": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// IDでユーザーを取得する
func getUserId(c *gin.Context) {
	user := findUser(c)
	if user != nil {
		c.JSON(http.StatusOK, user)
		return
	}
}

// すべてのユーザーを取得する
func getUserAll(c *gin.Context) {
	iter := spannerClient.ReadOnlyTransaction().Read(context.Background(), "Users", spanner.AllKeys(), []string{"id", "name", "money"})
	defer iter.Stop()

	users := []User{}
	err := iter.Do(func(row *spanner.Row) error {
		var user User
		if err := row.ToStruct(&user); err != nil {
			return err
		}
		users = append(users, user)
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// IDでユーザー情報を更新する
func updateUser(c *gin.Context) {
	var responseUser User
	if err := c.BindJSON(&responseUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ユーザーデータを更新する
	_, err := spannerClient.ReadWriteTransaction(context.Background(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: "UPDATE Users SET name = @name, money = @money WHERE id = @id",
			Params: map[string]interface{}{
				"id":    responseUser.ID,
				"name":  responseUser.Name,
				"money": responseUser.Money,
			},
		}
		count, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		} else if count == 0 {
			return fmt.Errorf("update user count: 0")
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, responseUser)
}

// IDでユーザーを削除する
func deleteUser(c *gin.Context) {
	user := findUser(c)
	if user == nil {
		return
	}

	// ユーザーデータを更新する
	_, err := spannerClient.ReadWriteTransaction(context.Background(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: "DELETE FROM Users WHERE id = @id",
			Params: map[string]interface{}{
				"id": user.ID,
			},
		}
		_, err := txn.Update(ctx, stmt)
		if err != nil {
			return fmt.Errorf("Failed to delete user: %w", err)
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// IDでユーザー取得する
func findUser(c *gin.Context) *User {
	var responseUser User
	if err := c.BindJSON(&responseUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	key := spanner.Key{responseUser.ID}

	row, getUserErr := spannerClient.ReadOnlyTransaction().ReadRow(context.Background(), "Users", key, []string{"id", "name", "money"})
	if getUserErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not found user id", "data": getUserErr.Error()})
		return nil
	}

	var postUser User
	if err := row.ToStruct(&postUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not found user id", "data": err.Error()})
		return nil
	}
	return &postUser
}
