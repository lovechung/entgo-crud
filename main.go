package main

import (
	"context"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	"github.com/lovechung/ent-test/ent"
	"github.com/lovechung/ent-test/ent/car"
	"github.com/lovechung/ent-test/ent/predicate"
	"github.com/lovechung/ent-test/ent/user"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	router := gin.Default()

	client, err := ent.Open(dialect.MySQL, "root:password@tcp(127.0.0.1:3306)/test?parseTime=true")
	if err != nil {
		log.Fatalf("mysql 连接失败: %v", err)
	}
	client = client.Debug()
	defer client.Close()
	ctx := context.Background()
	// 自动创建表
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// ======================== 用户api ========================
	// 分页查询用户
	router.GET("/user/list", func(ctx *gin.Context) {
		var page int
		pageVal := ctx.Query("page")
		if pageVal == "" {
			page = 1
		} else {
			page, _ = strconv.Atoi(pageVal)
		}

		var pageSize int
		pageSizeVal := ctx.Query("pageSize")
		if pageSizeVal == "" {
			pageSize = 5
		} else {
			pageSize, _ = strconv.Atoi(pageSizeVal)
		}

		username := ctx.Query("username")
		from := ctx.Query("from")
		to := ctx.Query("to")

		list, total := ListUser(ctx, client, page, pageSize, username, Str2Time(from), Str2Time(to))
		ctx.JSON(http.StatusOK, gin.H{"list": list, "total": total})
	})

	// 按id查询单个用户
	router.GET("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))
		result := GetUser(ctx, client, int64(id))
		ctx.JSON(http.StatusOK, result)
	})

	// 新增用户
	router.POST("/user", func(ctx *gin.Context) {
		username := ctx.Query("username")
		password := ctx.Query("password")
		result := CreateUser(ctx, client, username, password)
		ctx.JSON(http.StatusOK, result)
	})

	// 更新用户
	router.PUT("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))
		username := ctx.Query("username")
		password := ctx.Query("password")
		UpdateUser(ctx, client, int64(id), username, password)
		ctx.JSON(http.StatusOK, "")
	})

	// 删除用户
	router.DELETE("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Query("id"))
		DeleteUser(ctx, client, int64(id))
		ctx.JSON(http.StatusOK, "")
	})

	// ======================== 汽车api ========================
	// 查询某用户的所有汽车
	router.GET("/car/:userId", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Param("userId"))
		result := QueryCars(ctx, client, int64(userId))
		ctx.JSON(http.StatusOK, result)
	})

	// 查询某用户的所有汽车(包含用户信息)
	router.GET("/car-by-user/:userId", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Param("userId"))
		result := QueryCarsByUser(ctx, client, int64(userId))
		ctx.JSON(http.StatusOK, result)
	})

	// 新增汽车，并与用户绑定
	router.POST("/car", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Query("userId"))
		model := ctx.Query("model")
		CreateCars(ctx, client, int64(userId), model)
		ctx.JSON(http.StatusOK, "")
	})

	_ = router.Run()
}

// ============================= user相关 =============================

func ListUser(ctx context.Context, client *ent.Client,
	page int, pageSize int, username string, from time.Time, to time.Time) ([]*ent.User, int) {

	q := client.User.Query()

	// 组装查询条件
	cond := make([]predicate.User, 0)
	if username != "" {
		cond = append(cond, user.UsernameHasPrefix(username))
	}
	if !from.IsZero() {
		cond = append(cond, user.CreatedAtGTE(from))
	}
	if !to.IsZero() {
		cond = append(cond, user.CreatedAtLTE(to))
	}
	if len(cond) > 0 {
		q.Where(cond...)
	}

	// 查询总数
	total := q.CountX(ctx)

	// 查询列表
	list := q.Offset(GetPageOffset(page, pageSize)).
		Limit(pageSize).
		Order(ent.Desc(user.FieldCreatedAt)).
		AllX(ctx)

	return list, total
}

func GetUser(ctx context.Context, client *ent.Client, id int64) *ent.User {
	return client.User.GetX(ctx, id)
}

func CreateUser(ctx context.Context, client *ent.Client, username string, password string) *ent.User {
	return client.User.
		Create().
		SetUsername(username).
		SetPassword(password).
		SaveX(ctx)
}

func UpdateUser(ctx context.Context, client *ent.Client, id int64, username string, password string) {
	u := client.User.Update().Where(user.ID(id))
	if username != "" {
		u.SetUsername(username)
	}
	if password != "" {
		u.SetPassword(password)
	}
	u.ExecX(ctx)
}

func DeleteUser(ctx context.Context, client *ent.Client, id int64) {
	client.User.
		DeleteOneID(id).
		ExecX(ctx)
}

// ============================= car相关 =============================

func QueryCars(ctx context.Context, client *ent.Client, userId int64) []*ent.Car {
	// 最简单的单表查询，返回单表的字段
	return client.Car.
		Query().
		Where(car.UserID(userId)).
		AllX(ctx)
}

func QueryCarsByUser(ctx context.Context, client *ent.Client, userId int64) []*ent.User {
	// 使用图查询
	return client.User.
		Query().
		Where(user.ID(userId)).
		WithCars().
		AllX(ctx)
}

type CarResult struct {
	ent.User
	Model        string `sql:"model"`
	RegisteredAt string `sql:"registered_at"`
}

func QueryCarsModify(ctx context.Context, client *ent.Client, userId int64) []*CarResult {
	var rsp []*CarResult
	// 使用modifier查询(用于某些特定场景)
	client.User.Query().
		Where(user.ID(userId)).
		Modify(func(s *sql.Selector) {
			t := sql.Table(car.Table)
			s.LeftJoin(t).
				On(
					s.C(user.FieldID),
					t.C(car.FieldUserID),
				).
				AppendSelect(t.C(car.FieldModel), t.C(car.FieldRegisteredAt))
		}).
		ScanX(ctx, &rsp)
	return rsp
}

func CreateCars(ctx context.Context, client *ent.Client, userId int64, model string) {
	client.Car.
		Create().
		SetModel(model).
		SetRegisteredAt(time.Now()).
		SetUserID(userId).
		SaveX(ctx)
}

// ============================= 工具 =============================

func GetPageOffset(pageNum, pageSize int) int {
	return (pageNum - 1) * pageSize
}

func Time2Str(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func Str2Time(s string) time.Time {
	loc, _ := time.LoadLocation("Local")
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	return t
}
