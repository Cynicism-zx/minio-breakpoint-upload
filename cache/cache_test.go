package cache

import "testing"

func TestGet(t *testing.T) {
	data, err := Cache.Get("md5")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
}
