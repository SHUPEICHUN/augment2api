package api

import (
	"augment2api/config"
	"context"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TokenInfo 存储token信息
type TokenInfo struct {
	Token     string `json:"token"`
	TenantURL string `json:"tenant_url"`
}

// GetRedisTokenHandler 从Redis获取token列表
func GetRedisTokenHandler(c *gin.Context) {
	// 获取所有token的key
	tokenKeys, err := config.RDB.Keys(context.Background(), "token:*").Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "error",
			"error":  "获取token列表失败: " + err.Error(),
		})
		return
	}

	// 如果没有token
	if len(tokenKeys) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"tokens": []TokenInfo{},
		})
		return
	}

	// 构建token列表
	var tokenList []TokenInfo
	for _, key := range tokenKeys {
		// 从key中提取token (格式: "token:{token}")
		token := key[6:] // 去掉前缀 "token:"

		// 获取对应的tenant_url
		tenantURL, err := config.RDB.HGet(context.Background(), key, "tenant_url").Result()
		if err != nil {
			continue // 跳过无效的token
		}

		tokenList = append(tokenList, TokenInfo{
			Token:     token,
			TenantURL: tenantURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"tokens": tokenList,
	})
}

// SaveTokenToRedis 保存token到Redis
func SaveTokenToRedis(token, tenantURL string) error {
	// 创建一个唯一的key，包含token和tenant_url
	tokenKey := "token:" + token

	// 将tenant_url存储在token对应的哈希表中
	if err := config.RDB.HSet(context.Background(), tokenKey, "tenant_url", tenantURL).Err(); err != nil {
		return err
	}

	return nil
}

// GetRandomToken 从Redis中随机获取一个token
func GetRandomToken() (string, string) {
	// 获取所有token的key
	tokenKeys, err := config.RDB.Keys(context.Background(), "token:*").Result()
	if err != nil || len(tokenKeys) == 0 {
		return "", ""
	}

	// 随机选择一个token
	randomIndex := rand.Intn(len(tokenKeys))
	randomKey := tokenKeys[randomIndex]

	// 从key中提取token
	token := randomKey[6:] // 去掉前缀 "token:"

	// 获取对应的tenant_url
	tenantURL, err := config.RDB.HGet(context.Background(), randomKey, "tenant_url").Result()
	if err != nil {
		return "", ""
	}

	return token, tenantURL
}

// DeleteTokenHandler 删除指定的token
func DeleteTokenHandler(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "未指定token",
		})
		return
	}

	tokenKey := "token:" + token

	// 检查token是否存在
	exists, err := config.RDB.Exists(context.Background(), tokenKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "检查token失败: " + err.Error(),
		})
		return
	}

	if exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "token不存在",
		})
		return
	}

	// 删除token
	if err := config.RDB.Del(context.Background(), tokenKey).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "删除token失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

// UseTokenHandler 设置指定的token为当前活跃token
func UseTokenHandler(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "未指定token",
		})
		return
	}

	tokenKey := "token:" + token

	// 检查token是否存在
	exists, err := config.RDB.Exists(context.Background(), tokenKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "检查token失败: " + err.Error(),
		})
		return
	}

	if exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "token不存在",
		})
		return
	}

	// 设置当前活跃token
	if err := config.RedisSet("current_token", token, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "设置当前token失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}
