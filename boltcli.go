package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/urfave/cli"
)

func main() {
	var file string
	var action string
	var key string
	var value string
	var bucket string

	cli.AppHelpTemplate = `
{{.Name}} - {{.Usage}}

VERSION:
  {{.Version}}

USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}

GLOBAL OPTIONS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

INSTALLATION:

  go install github.com/qiwihui/boltcli

EXAMPLES:

  Please install jq first.

  1. $GOPATH/bin/boltcli -f dbshield.db -c get | jq
  
  {
	"return_code": 0,
	"message": "success",
	"data": [
	  "abnormal",
	  "pattern",
	  "state"
	]
  }

  2. $GOPATH/bin/boltcli -f dbshield.db -c get -b pattern | jq

  {
	"return_code": 0,
	"message": "success",
	"data": [
	  {
		"key": "0x0000e0030000002a0000e0076669727374",
		"value": "select * from first"
	  },
	  {
		"key": "0x0000e0030000002a0000e00766697273740000e0086e616d650000003c0000e023",
		"value": "select * from first where name<100"
	  },
	  {
		"key": "0x0000e003404076657273696f6e5f636f6d6d656e740000e00d0000e023",
		"value": "select @@version_comment limit 1"
	  }
	]
  }

AUTHOR:
  {{range .Authors}}{{ . }}{{end}}
`
	app := cli.NewApp()
	app.Name = "boltcli"
	app.Usage = "view and update boltdb file in your terminal"
	app.Version = "1.0.0"
	app.Author = "qiwihui"
	app.Email = "qwh005007@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "file, f",
			Usage:       "boltdb `FILE` to view and update",
			Destination: &file,
		},
		cli.StringFlag{
			Name:        "action, c",
			Usage:       "action to update boltdb: get(default), set, delete",
			Destination: &action,
		},
		cli.StringFlag{
			Name:        "bucket, b",
			Usage:       "boltdb `BUCKET` to view",
			Destination: &bucket,
		},
		cli.StringFlag{
			Name:        "key, k",
			Usage:       "boltdb `KEY` to view",
			Destination: &key,
		},
		cli.StringFlag{
			Name:        "value, r",
			Usage:       "boltdb `VALUE` to set",
			Destination: &value,
		},
	}
	app.Action = func(c *cli.Context) error {

		context := Context{}
		if file == "" {
			context.ReturnCode = -1
			context.Message = "No file input"
			return context.Print()
		}

		// check if db file exist
		if _, err := os.Stat(file); os.IsNotExist(err) {
			context.ReturnCode = -1
			context.Message = "DB file not found!"
			return context.Print()
		}

		failedOperationContext := Context{ReturnCode: -1, Message: "failed"}
		succeedOperationContext := Context{ReturnCode: 0, Message: "success"}

		// fmt.Println("action: ", action)
		// fmt.Println("bucket: ", bucket)
		// fmt.Println("key: ", key)
		// fmt.Println("value: ", value)

		switch action {
		case "set":
			if bucket != "" && key != "" && value != "" {
				err := updateBucketKey(file, bucket, key, value)
				if err != nil {
					return succeedOperationContext.Print()
				}
			}
			return failedOperationContext.Print()
		case "delete":
			if bucket != "" && key != "" {
				err := deleteBucketKey(file, bucket, key)
				if err == nil {
					return succeedOperationContext.Print()
				}
			}
			return failedOperationContext.Print()
		default:
			if bucket != "" {
				if key != "" {
					data := getBucketKeyValue(file, bucket, key)
					if data != nil {
						succeedOperationContext.Data = data
						return succeedOperationContext.Print()
					}
				} else {
					data := getBucketKeys(file, bucket)
					if data != nil {
						succeedOperationContext.Data = data
						return succeedOperationContext.Print()
					}
				}
			} else {
				data := getBuckets(file)
				if data != nil {
					succeedOperationContext.Data = data
					return succeedOperationContext.Print()
				}
			}
			return failedOperationContext.Print()
		}
	}
	app.Run(os.Args)
}

// Context return data
type Context struct {
	ReturnCode int         `json:"return_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

// Print print to console
func (context *Context) Print() error {
	// fmt.Println("==> context: ", context)
	b, err := json.Marshal(context)
	if err != nil {
		return err
	}
	os.Stdout.Write(b)
	return nil
}

// 获取数据库
func getDb(file string) *bolt.DB {
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return db
}

// 获取Bucket列表
func getBuckets(file string) interface{} {
	db := getDb(file)
	defer db.Close()

	bucketsList := []string{}
	db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			bucketsList = append(bucketsList, string(name))
			return nil
		})
		return nil
	})
	return bucketsList
}

// 获取全部键值
func getBucketKeys(file string, bucket string) interface{} {
	db := getDb(file)
	defer db.Close()

	type Pattern struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	patterns := []Pattern{}

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("pattern"))
		if b != nil {
			b.ForEach(func(k, v []byte) error {
				if strings.Index(string(k), "_client_") == -1 && strings.Index(string(k), "_user_") == -1 {
					var sx16 = fmt.Sprintf("0x%x", k)
					patterns = append(patterns, Pattern{Key: sx16, Value: string(v)})
				}
				return nil
			})

		}
		return nil
	})
	return patterns
}

func getBucketKeyValue(file string, bucket string, key string) interface{} {
	db := getDb(file)
	defer db.Close()

	var returnValue []byte

	db.View(func(tx *bolt.Tx) error {
		curBucket := tx.Bucket([]byte(bucket))
		if strings.Index(string(key), "_client_") == -1 && strings.Index(string(key), "_user_") == -1 {
			value := curBucket.Get([]byte(key))
			returnValue = value
		}
		return nil
	})
	return string(returnValue)
}

// 删除 key
func deleteBucketKey(file string, bucket string, key string) error {
	db := getDb(file)
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			return b.Delete([]byte(key))
		}
		return nil
	})
}

// 更新key
func updateBucketKey(file string, bucket string, key string, value string) error {
	db := getDb(file)
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			return b.Put([]byte(key), []byte(value))
		}
		return nil
	})
}
