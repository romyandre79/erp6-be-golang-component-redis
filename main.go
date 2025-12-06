package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Input struct {
	Params []struct {
		InputName string `json:"inputname"`
		CompValue string `json:"compvalue"`
	} `json:"params"`
}

type Output struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

func main() {
	var input Input
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("failed to decode input: %v", err)})
		return
	}

	var (
		addr       string
		password   string
		db         int
		action     = "get"
		key        string
		value      string
		expiration int
	)

	// Extract parameters
	for _, p := range input.Params {
		val := strings.TrimSpace(p.CompValue)
		switch strings.ToLower(p.InputName) {
		case "addr":
			addr = val
		case "password":
			password = val
		case "db":
			fmt.Sscanf(val, "%d", &db)
		case "action":
			if val != "" {
				action = strings.ToLower(val)
			}
		case "key":
			key = val
		case "value":
			value = val
		case "expiration":
			fmt.Sscanf(val, "%d", &expiration)
		}
	}

	// Validate required parameters
	if addr == "" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "addr is required"})
		return
	}
	if key == "" && action != "keys" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "key is required"})
		return
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})

	ctx := context.Background()
	var result interface{}
	var err error

	switch action {
	case "set":
		err = rdb.Set(ctx, key, value, time.Duration(expiration)*time.Second).Err()
		if err == nil {
			result = "OK"
		}
	case "get":
		result, err = rdb.Get(ctx, key).Result()
	case "del":
		var count int64
		count, err = rdb.Del(ctx, key).Result()
		result = count
	case "exists":
		var count int64
		count, err = rdb.Exists(ctx, key).Result()
		result = count > 0 // Return boolean
	case "keys":
		// key param works as pattern for keys
		pattern := key
		if pattern == "" {
			pattern = "*"
		}
		result, err = rdb.Keys(ctx, pattern).Result()
	default:
		json.NewEncoder(os.Stdout).Encode(Output{Error: "invalid action"})
		return
	}

	if err != nil {
		if err == redis.Nil {
			json.NewEncoder(os.Stdout).Encode(Output{Result: nil}) // Key not found
		} else {
			json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("redis error: %v", err)})
		}
	} else {
		json.NewEncoder(os.Stdout).Encode(Output{Result: result})
	}
}
