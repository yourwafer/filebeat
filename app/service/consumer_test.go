package service

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateFormat(t *testing.T) {
	now := time.Now()
	laylout := "2006/01-02 15:04:05.000"
	format := now.Format(laylout)
	mill := now.UnixNano() / int64(time.Millisecond)
	after := time.Unix(0, mill*int64(time.Millisecond))
	afterFormat := after.Format(laylout)
	if afterFormat != format {
		t.Fail()
	}
}

func TestJsonBoolean(t *testing.T) {
	value := map[string]interface{}{
		"charge": true,
	}
	jsonVal, _ := json.Marshal(value)
	if "{\"charge\":true}" != string(jsonVal) {
		t.Fail()
	}
}
