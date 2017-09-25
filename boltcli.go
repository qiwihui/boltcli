package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
	"github.com/xwb1989/sqlparser"
)

func main() {
	var file string
	var config string
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

  1. $GOPATH/bin/boltcli -f dbshield.db -t get | jq
  
  {
	"return_code": 0,
	"message": "success",
	"data": [
	  "abnormal",
	  "pattern",
	  "state"
	]
  }

  2. $GOPATH/bin/boltcli -f dbshield.db -t get -b pattern | jq

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
			Name:        "config, c",
			Usage:       "DBShield config file",
			Destination: &config,
		},
		cli.StringFlag{
			Name:        "action, t",
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
			Value:       "",
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
		var dbLocation string
		if file == "" && config == "" {
			context.ReturnCode = -1
			context.Message = "No DBShield config file or boltdb file input"
			return context.Print()
		} else if config != "" {
			if _, err := os.Stat(config); os.IsNotExist(err) {
				context.ReturnCode = -1
				context.Message = "DBShield config file not found!"
				return context.Print()
			}
			viper.SetConfigFile(config)
			// Read the config file
			err := viper.ReadInConfig()
			if err != nil {
				context.ReturnCode = -1
				context.Message = "DBShield config file read error!"
				return context.Print()
			}
			TargetIP, err := strConfig("targetIP")
			if err != nil {
				return err
			}
			DBType := strConfigDefualt("dbms", "mysql")
			DBDir := strConfigDefualt("dbDir", os.TempDir()+"/model")
			dbLocation = path.Join(DBDir, TargetIP+"_"+DBType) + ".db"
		} else {
			dbLocation = file
		}
		// check if db file exist, check DBShield config first
		if _, err := os.Stat(dbLocation); os.IsNotExist(err) {
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
				err := updateBucketKey(dbLocation, bucket, key, value)
				if err != nil {
					return succeedOperationContext.Print()
				}
			}
			return failedOperationContext.Print()
		case "delete":
			if bucket != "" && key != "" {
				err := deleteBucketKey(dbLocation, bucket, key)
				if err == nil {
					return succeedOperationContext.Print()
				}
			}
			return failedOperationContext.Print()
		default:
			if bucket != "" {
				if key != "" {
					data := getBucketKeyValue(dbLocation, bucket, key)
					if data != nil {
						succeedOperationContext.Data = data
						return succeedOperationContext.Print()
					}
				} else {
					data := getBucketKeys(dbLocation, bucket)
					if data != nil {
						succeedOperationContext.Data = data
						return succeedOperationContext.Print()
					}
				}
			} else {
				// fmt.Println("get bucketlist")
				data := getBuckets(dbLocation)
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

func strConfig(key string) (ret string, err error) {
	if viper.IsSet(key) {
		ret = viper.GetString(key)
		return
	}
	// err = fmt.Errorf("Invalid '%s' cofiguration", key)
	return
}

func strConfigDefualt(key, defaultValue string) (ret string) {
	if viper.IsSet(key) {
		ret = viper.GetString(key)
		return
	}
	// logger.Infof("'%s' not configured, assuming: %s", key, defaultValue)
	ret = defaultValue
	return
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

	dst, err := formatStringToKey(key)
	if err != nil {
		return nil
	}

	returnValue := []byte{}
	db.View(func(tx *bolt.Tx) error {
		curBucket := tx.Bucket([]byte(bucket))
		if curBucket != nil {
			if strings.Index(string(dst), "_client_") == -1 && strings.Index(string(dst), "_user_") == -1 {
				value := curBucket.Get(dst)
				returnValue = value
			}
		}
		return nil
	})
	return string(returnValue)
}

// 删除 key
func deleteBucketKey(file string, bucket string, key string) error {
	db := getDb(file)
	defer db.Close()

	dst, err := formatStringToKey(key)
	if err != nil {
		return nil
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			return b.Delete(dst)
		}
		return nil
	})
}

func formatStringToKey(key string) (dst []byte, err error) {
	if !strings.HasPrefix(key, "0x") {
		return []byte{}, errors.New("wrong prefix")
	}
	akey := []byte(key[2:])
	dst = make([]byte, hex.DecodedLen(len(akey)))
	hex.Decode(dst, akey)
	return dst, nil
}

// 更新key
func updateBucketKey(file string, bucket string, key string, value string) error {
	db := getDb(file)
	defer db.Close()

	dst, err := formatStringToKey(key)
	if err != nil {
		return nil
	}

	patternOfValue := Pattern(value)

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			if b != nil {
				b.Delete(dst)
				return b.Put(patternOfValue, []byte(value))
			}
			return nil
		}
		return nil
	})
}

//Pattern returns pattern of given query
func Pattern(query string) []byte {
	tokenizer := sqlparser.NewStringTokenizer(query)
	buf := bytes.Buffer{}
	l := make([]byte, 4)
	for {
		typ, val := tokenizer.Scan()
		switch typ {
		case sqlparser.ID: //table, database, variable & ... names
			buf.Write(val)
		case 0: //End of query
			return buf.Bytes()
		default:
			binary.BigEndian.PutUint32(l, uint32(typ))
			buf.Write(l)
		}
	}
}
