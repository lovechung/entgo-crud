package main

import (
	"context"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	"github.com/lovechung/ent-crud/biz"
	"github.com/lovechung/ent-crud/ent"
	"github.com/lovechung/ent-crud/ent/car"
	"github.com/lovechung/ent-crud/ent/predicate"
	"github.com/lovechung/ent-crud/ent/user"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	router := gin.Default()

	client, err := ent.Open(dialect.MySQL, "root:Jicco_2021@tcp(192.168.31.96:3306)/test?parseTime=true")
	if err != nil {
		log.Fatalf("mysql 连接失败: %v", err)
	}
	client = client.Debug()
	defer client.Close()
	context.Background()
	// 自动创建表
	//if err := client.Schema.Create(ctx); err != nil {
	//	log.Fatalf("failed creating schema resources: %v", err)
	//}

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

	// 按id查询单个用户全字段
	router.GET("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))
		result := GetUser(ctx, client, int64(id))
		ctx.JSON(http.StatusOK, result)
	})

	// 按姓名查询单个用户部分字段
	router.GET("/user-by-name", func(ctx *gin.Context) {
		username := ctx.Query("username")
		result := GetUserByName(ctx, client, username)
		ctx.JSON(http.StatusOK, result)
	})

	// 新增用户
	router.POST("/user", func(ctx *gin.Context) {
		username := ctx.Query("username")
		password := ctx.Query("password")
		result := CreateUser(ctx, client, &biz.User{
			Username: &username,
			Password: &password,
		})
		ctx.JSON(http.StatusOK, result)
	})

	// 更新用户
	router.PUT("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))

		// 模拟protobuf的optional功能
		var username *string
		usernameVal := ctx.Query("username")
		if usernameVal != "" {
			username = &usernameVal
		}

		// 模拟protobuf的optional功能
		var password *string
		passwordVal := ctx.Query("password")
		if passwordVal != "" {
			password = &passwordVal
		}

		UpdateUser(ctx, client, &biz.User{
			Id:       int64(id),
			Username: username,
			Password: password,
		})
		ctx.JSON(http.StatusOK, "更新用户成功")
	})

	// 删除用户
	router.DELETE("/user/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))
		DeleteUser(ctx, client, int64(id))
		ctx.JSON(http.StatusOK, "删除用户成功")
	})

	// 查询某用户的所有汽车
	router.GET("/user/:id/cars", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Param("id"))
		result := QueryCarsByUser(ctx, client, int64(userId))
		ctx.JSON(http.StatusOK, result)
	})

	// 查询某用户的所有汽车(未聚合)
	router.GET("/user/:id/cars2", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Param("id"))
		result := QueryCarsByUser2(ctx, client, int64(userId))
		ctx.JSON(http.StatusOK, result)
	})

	// ======================== 汽车api ========================
	// 按条件查询汽车
	router.GET("/car/list", func(ctx *gin.Context) {
		var userId *int64
		userIdVal := ctx.Query("userId")
		if userIdVal != "" {
			i, _ := strconv.Atoi(userIdVal)
			z := int64(i)
			userId = &z
		}

		var model *string
		modelVal := ctx.Query("model")
		if modelVal != "" {
			model = &modelVal
		}

		result := QueryCars(ctx, client, userId, model)
		ctx.JSON(http.StatusOK, result)
	})

	// 新增汽车
	router.POST("/car", func(ctx *gin.Context) {
		userId, _ := strconv.Atoi(ctx.Query("userId"))
		model := ctx.Query("model")
		CreateCars(ctx, client, int64(userId), model)
		ctx.JSON(http.StatusOK, "新增汽车成功")
	})

	// 更新汽车
	router.PUT("/car/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))

		var userId *int64
		userIdVal := ctx.Query("userId")
		if userIdVal != "" {
			i, _ := strconv.Atoi(userIdVal)
			z := int64(i)
			userId = &z
		}

		var model *string
		modelVal := ctx.Query("model")
		if modelVal != "" {
			model = &modelVal
		}

		UpdateCar(ctx, client, &biz.Car{
			Id:     int64(id),
			UserId: userId,
			Model:  model,
		})
		ctx.JSON(http.StatusOK, "更新汽车成功")
	})

	// 删除汽车
	router.DELETE("/car/:id", func(ctx *gin.Context) {
		id, _ := strconv.Atoi(ctx.Param("id"))
		DeleteCar(ctx, client, int64(id))
		ctx.JSON(http.StatusOK, "删除汽车成功")
	})

	_ = router.Run()
}

// ============================= user相关 =============================

func ListUser(ctx context.Context, client *ent.Client,
	page, pageSize int, username string, from time.Time, to time.Time) ([]*ent.User, int) {

	q := client.User.Query()

	// 组装查询条件
	cond := make([]predicate.User, 0)
	if username != "" {
		cond = append(cond, user.UsernameContains(username))
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

func GetUserByName(ctx context.Context, client *ent.Client, username string) *biz.UserReply {
	u := client.User.
		Query().
		Where(user.Username(username)).
		Select(user.FieldID, user.FieldUsername, user.FieldPassword).
		OnlyX(ctx)
	return &biz.UserReply{
		Id:       u.ID,
		Username: u.Username,
		Password: u.Password,
	}
}

func CreateUser(ctx context.Context, client *ent.Client, u *biz.User) *ent.User {
	return client.User.
		Create().
		SetUser(u).
		SaveX(ctx)
}

func UpdateUser(ctx context.Context, client *ent.Client, u *biz.User) {
	client.User.
		Update().
		Where(user.ID(u.Id)).
		SetUser(u).
		ExecX(ctx)
}

func DeleteUser(ctx context.Context, client *ent.Client, id int64) {
	client.User.
		DeleteOneID(id).
		ExecX(ctx)
}

func QueryCarsByUser(ctx context.Context, client *ent.Client, userId int64) *biz.UserCarsReply {
	// 使用图查询
	u := client.User.
		Query().
		Where(user.ID(userId)).
		WithCars().
		OnlyX(ctx)

	cars := u.Edges.Cars
	list := make([]*biz.CarReply, 0)
	for _, c := range cars {
		list = append(list, &biz.CarReply{
			Id:           c.ID,
			Model:        c.Model,
			RegisteredAt: Time2Str(c.RegisteredAt),
		})
	}

	return &biz.UserCarsReply{
		Id:        u.ID,
		Username:  u.Username,
		CreatedAt: Time2Str(u.CreatedAt),
		Cars:      list,
	}
}

type UserCarsReply2 struct {
	ent.User
	Model        string `sql:"model"`
	RegisteredAt string `sql:"registered_at"`
}

func QueryCarsByUser2(ctx context.Context, client *ent.Client, userId int64) []*UserCarsReply2 {
	var rsp []*UserCarsReply2
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

// ============================= car相关 =============================

func QueryCars(ctx context.Context, client *ent.Client, userId *int64, model *string) []*ent.Car {
	q := client.Car.Query()

	// 组装查询条件
	cond := make([]predicate.Car, 0)
	if userId != nil {
		cond = append(cond, car.UserID(*userId))
	}
	if model != nil {
		cond = append(cond, car.ModelContains(*model))
	}
	if len(cond) > 0 {
		q.Where(cond...)
	}

	return q.AllX(ctx)
}

func CreateCars(ctx context.Context, client *ent.Client, userId int64, model string) {
	client.Car.
		Create().
		SetModel(model).
		SetRegisteredAt(time.Now()).
		SetUserID(userId).
		SaveX(ctx)
}

func UpdateCar(ctx context.Context, client *ent.Client, c *biz.Car) {
	client.Car.
		Update().
		Where(car.ID(c.Id)).
		SetCar(c).
		ExecX(ctx)
}

func DeleteCar(ctx context.Context, client *ent.Client, id int64) {
	client.Car.
		DeleteOneID(id).
		ExecX(ctx)
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
