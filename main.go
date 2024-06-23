package main

import (
	"encoding/json"
	"fmt"
	"log"
	"miHttpServer/middlewares"
	"miHttpServer/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

/*
其实这个响应结构体不是必须的
因为map的迭代顺序是不确定的，可能会出现msg在data下面的情况
因此为了统一格式，定义一个响应结构体
*/
type ResponseData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 增加商品信息
func AddData(ctx *gin.Context) {
	data, _ := ctx.GetRawData()
	var response ResponseData
	var jsonStr map[string]interface{}
	err := json.Unmarshal(data, &jsonStr)
	// 这个只能判断json的key类型是否正确，不能判断value的类型是否正确
	if err != nil {
		response.Code = 1
		response.Msg = "客户端传递的json非法"
		response.Data = err.Error()
		ctx.JSON(http.StatusBadRequest, response)
	} else {
		// 如果json的value类型不正确，会导致panic
		defer func() {
			if err := recover(); err != nil {
				response.Code = 1
				response.Msg = "客户端传递的json值无效"
				response.Data = err.(error).Error()
				ctx.JSON(http.StatusInternalServerError, response)
			}
		}()
		item := models.Item{
			Name:  jsonStr["name"].(string),
			Price: jsonStr["price"].(float64),
		}
		_, err = models.InsertItem(&item)
		if err != nil {
			response.Code = 1
			response.Msg = "插入数据失败"
			response.Data = err.Error()
			ctx.JSON(http.StatusInternalServerError, response)
		} else {
			item_info := make(map[string]interface{})
			item_info["item_info"] = map[string]interface{}{
				"item_id": item.ItemID,
				"name":    item.Name,
				"price":   item.Price,
			}
			response.Code = 0
			response.Msg = "成功"
			response.Data = item_info
			ctx.JSON(http.StatusOK, response)
		}
	}
}

// 修改商品信息
func UpdateData(ctx *gin.Context) {
	itemIDStr := ctx.Param("item_id")
	var response ResponseData
	item_id, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		response.Code = 1
		response.Msg = "链接中的item_id非法"
		response.Data = err.Error()
		ctx.JSON(http.StatusBadRequest, response)
	} else {
		data, _ := ctx.GetRawData()
		var jsonStr map[string]interface{}
		err = json.Unmarshal(data, &jsonStr)
		if err != nil {
			response.Code = 1
			response.Msg = "客户端传递的json非法"
			response.Data = err.Error()
			ctx.JSON(http.StatusBadRequest, response)
		} else {
			defer func() {
				if err := recover(); err != nil {
					response.Code = 1
					response.Msg = "客户端传递的json值无效"
					response.Data = err.(error).Error()
					ctx.JSON(http.StatusInternalServerError, response)
				}
			}()
			item := models.Item{
				ItemID: item_id,
				Name:   jsonStr["name"].(string),
				Price:  jsonStr["price"].(float64),
			}
			n, err := models.UpdateItem(item_id, &item)
			if err != nil {
				response.Code = 1
				response.Msg = "更新数据失败，item_id：" + itemIDStr
				response.Data = err.Error()
				ctx.JSON(http.StatusInternalServerError, response)
			} else if n == 0 {
				response.Code = 1
				response.Msg = "未找到相关记录"
				response.Data = fmt.Sprintf("item_id为%v的商品不存在", item_id)
				ctx.JSON(http.StatusInternalServerError, response)
			} else {
				store_info := make(map[string]interface{})
				store_info["store_info"] = map[string]interface{}{
					"item_id": item.ItemID,
					"name":    item.Name,
					"price":   item.Price,
				}
				response.Code = 0
				response.Msg = "成功"
				response.Data = store_info
				ctx.JSON(http.StatusOK, response)
			}
		}
	}
}

func main() {
	// 设置gin的运行模式，ReleaseMode表示生产模式，不显示日志的调试信息
	gin.SetMode(gin.ReleaseMode)
	// 设置日志文件
	ginLogFile, err := os.Create("./logs/miHttpServer.log")
	if err != nil {
		log.Printf("无法创建日志文件: %v\n", err)
	}
	defer ginLogFile.Close()

	// 创建一个服务（不使用默认的中间件）
	ginServer := gin.New()
	//使用自定义的控制台日志格式
	ginServer.Use(gin.LoggerWithFormatter(middlewares.CustomConsoleLogger))

	// 自定义文件日志输出格式
	customFileLogger := func(params gin.LogFormatterParams) string {
		return fmt.Sprintf(
			"[miHttpServer] %s |%d|  %s	 %s\n",
			params.TimeStamp.Format("2006-01-02 15:04:05"),
			params.StatusCode,
			params.Method,
			params.Path,
		)
	}
	// 使用自定义的文件日志格式
	ginServer.Use(func(ctx *gin.Context) {
		// 确保按顺序执行中间件链中的下一个中间件
		ctx.Next()
		param := gin.LogFormatterParams{
			TimeStamp:  time.Now(),
			StatusCode: ctx.Writer.Status(),
			Method:     ctx.Request.Method,
			Path:       ctx.Request.URL.Path,
		}
		// 记录日志到文件
		ginLogFile.WriteString(customFileLogger(param))
	})
	ginServer.Use(gin.Recovery())
	// 连接数据库
	err = models.InitDB()
	if err != nil {
		log.Fatalln("连接数据库失败:", err)
	}

	// 增加商品信息（从JSON获取）
	ginServer.PUT("/item", AddData)
	ginServer.POST("/item", AddData)

	// 修改商品信息
	ginServer.POST("/item/:item_id", UpdateData)

	// 查询商品信息
	ginServer.GET("item/:item_id", func(ctx *gin.Context) {
		response := ResponseData{
			Code: 0,
			Msg:  "成功",
			Data: "暂无数据",
		}
		ctx.JSON(http.StatusOK, response)
	})
	// 删除商品信息
	ginServer.DELETE("/item/:item_id", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "成功",
			"data": "暂无数据",
		})
	})
	err = ginServer.Run(":8080")
	if err != nil {
		log.Fatal("项目启动失败:", err)
	}
}
