package main

import (
	"EmulatorAndApiServerTest/sub"
	"flag"
	"os"

	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/gin-gonic/gin"
)

// User はテーブルのレコードを表す構造体
type User struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Money string  `json:"money"`
}

// spannerClient は Spanner クライアント
var spannerClient *spanner.Client

// dbName は Spanner データベース名
var dbName string

func main() {
	PortNumStr := ":8080" // デフォルトポート番号
	// 環境変数からポート番号を取得
	serverPort := os.Getenv("TEST_EMULATORS_API_SERVER_PORT")
	if serverPort != "" {
		PortNumStr = serverPort
	}
	
	// 変数でflagを定義
	var (
		p = flag.String("p", "memory-dev-3dc1b", "プロジェクトID")       // -p オプションでプロジェクトIDを指定する
		i = flag.String("i", "takahira-test-instance", "インスタンスID") // -i オプションでインスタンスIDを指定する
		d = flag.String("d", "takahira-test-databases", "データベース名") // -d オプションでデータベース名を指定する
	)
	// ここで解析
	flag.Parse()

	ctx := context.Background()

	// Spanner クライアントの初期化
	projectID := *p                                                                          // プロジェクトID
	instanceID := *i                                                                         // インスタンスID
	dbName = fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, *d) // データベース名
	fmt.Println(dbName)

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
	r.PUT("/users/update/addmoney", addUserMoney)
	r.DELETE("/users/delete", deleteUser)

	r.Run(PortNumStr)
}

// createUser は新しいユーザーを作成する
func createUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// log.Println("Received user data:", user) // ログ出力

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
			return err
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
	var responseUser User
	if err := c.BindJSON(&responseUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not bind JSON", "data": err.Error()})
		return
	}
	key := spanner.Key{responseUser.ID}
	
	row, err := spannerClient.Single().ReadRow(context.Background(), "Users", key, []string{"id", "name", "money"})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not found user id", "data": err.Error()})
		return
	}

	var postUser User
	if err := row.ToStruct(&postUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Not found user id", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, postUser)
}

// すべてのユーザーを取得する
func getUserAll(c *gin.Context) {
	iter := spannerClient.Single().Read(context.Background(), "Users", spanner.AllKeys(), []string{"id", "name", "money"})
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

// IDでユーザーの所持金を追加して更新する
func addUserMoney(c *gin.Context) {
	var responseUser User
	if err := c.BindJSON(&responseUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ユーザーデータを更新する
	_, err := spannerClient.ReadWriteTransaction(context.Background(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 現在の money の値を読み取る
		row, err := txn.ReadRow(ctx, "Users", spanner.Key{responseUser.ID}, []string{"money"})
		if err != nil {
			return err
		}
		var currentMoney string
		if err := row.Columns(&currentMoney); err != nil {
			return err
		}

		// 新しい money の値を計算
		if responseUser.Money, err = sub.AddNum(currentMoney, responseUser.Money); err != nil {
			return err
		}
		
		// 新しい money の値で更新
		stmt := spanner.Statement{
			SQL: "UPDATE Users SET money = @money WHERE id = @id",
			Params: map[string]interface{}{
				"id": responseUser.ID,
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
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ユーザーデータを削除する
	var deleteUserData User
	_, err := spannerClient.ReadWriteTransaction(context.Background(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Users", spanner.Key{user.ID}, []string{"id", "name", "money"})
		if err != nil {
			return err
		}
		
		if err := row.ToStruct(&deleteUserData); err != nil {
			return err
		}
		
		stmt := spanner.Statement{
			SQL: "DELETE FROM Users WHERE id = @id",
			Params: map[string]interface{}{
				"id": user.ID,
			},
		}
		count, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		} else if count == 0 {
			return fmt.Errorf("Not found user id")
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "deleteUser": deleteUserData})
}