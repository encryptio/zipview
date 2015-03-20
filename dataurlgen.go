package main

import (
	"encoding/base64"
)

func toDataURL(typ string, data []byte) string {
	return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(data)
}
