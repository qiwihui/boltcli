
boltcli - view and update boltdb file in your terminal

VERSION:
  1.0.0

USAGE:
  boltcli [global options]

GLOBAL OPTIONS:
  --file FILE, -f FILE        boltdb FILE to view and update
  --action value, -c value    action to update boltdb: get(default), set, delete
  --bucket BUCKET, -b BUCKET  boltdb BUCKET to view
  --key KEY, -k KEY           boltdb KEY to view
  --value VALUE, -r VALUE     boltdb VALUE to set
  --help, -h                  show help
  --version, -v               print the version
  

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
  qiwihui <qwh005007@gmail.com>
