package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFlakeUid(t *testing.T) {
	id := GetFlakeUid()
	assert.NotEqual(t, 0, id)
}

func TestGetFlakeUidStr(t *testing.T) {
	id := GetFlakeUidStr()
	assert.NotEqual(t, "", id)
	assert.Equal(t, 18, len(id))
}

func TestGetLocalIPs(t *testing.T) {
	_, err := getLocalIPs()
	assert.Equal(t, nil, err)
}

func TestSumIPs_with_nil(t *testing.T) {
	total := sumIPs(nil)
	assert.Equal(t, uint16(0), total)
}
