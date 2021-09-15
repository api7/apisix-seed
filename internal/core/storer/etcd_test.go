package storer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
)

var host = "localhost:2379" // nolint:unused

func TestEtcdV3(t *testing.T) {
	// Make sure that etcd is installed in your test environment
	// Then comment out the following statement
	t.SkipNow()

	client, err := NewEtcd(conf.Etcd{Host: []string{host}})
	assert.Nil(t, err, "Test create etcd client")

	testCommon(t, client)
	testList(t, client)
	testWatch(t, client)

	err = client.Close()
	assert.Nil(t, err, "Test close etcd client")
}

// nolint:unused
func testCommon(t *testing.T, client *EtcdV3) {
	value, newValue := "test_value", "new_test_value"
	for _, key := range []string{
		"/apisix",
		"/apisix/routes",
		"/apisix/routes/1",
	} {
		// Create the key/value
		err := client.Create(context.Background(), key, value)
		assert.Nil(t, err, "Test create key")

		// Get should return the value
		val, err := client.Get(context.Background(), key)
		assert.Nil(t, err, "Test key get")
		assert.Equal(t, value, val, "Test get key value")

		// Update the key/value
		err = client.Update(context.Background(), key, newValue)
		assert.Nil(t, err, "Test key update")

		// Get should return the new value
		val, err = client.Get(context.Background(), key)
		assert.Nil(t, err, "Test key get")
		assert.Equal(t, newValue, val, "Test get key: new value")

		// Delete the key
		err = client.Delete(context.Background(), key)
		assert.Nil(t, err, "Test key delete")

		// Delete the non-existing key
		err = client.Delete(context.Background(), key)
		wantErr := fmt.Errorf("key: %s is not found", key)
		assert.Equal(t, wantErr, err, "Test delete non-existing key")

		// Get should fail
		_, err = client.Get(context.Background(), key)
		assert.Equal(t, wantErr, err, "Test get non-existing key")
	}
}

// nolint:unused
func testList(t *testing.T, client *EtcdV3) {
	prefix := "testList"
	firstKey, firstValue := "testList/first", "first"
	secondKey, secondValue := "testList/second", "second"

	err := client.Create(context.Background(), firstKey, firstValue)
	assert.Nil(t, err)
	err = client.Create(context.Background(), secondKey, secondValue)
	assert.Nil(t, err)

	for _, parent := range []string{prefix, prefix + "/"} {
		pairs, err := client.List(context.Background(), parent)
		assert.Nil(t, err, "Test list prefix")
		assert.Len(t, pairs, 2, "Test list content")

		for _, pair := range pairs {
			switch pair.Key {
			case firstKey:
				assert.Equal(t, firstKey, pair.Value)
			case secondKey:
				assert.Equal(t, secondKey, pair.Value)
			}
		}
	}

	err = client.DeletePrefix(context.Background(), prefix)
	assert.Nil(t, err, "Test delete prefix")

	// List should fail
	wantErr := fmt.Errorf("prefix: %s is not found", prefix)
	pairs, err := client.List(context.Background(), prefix)
	assert.Equal(t, wantErr, err, "Test list non-existing prefix")
	assert.Nil(t, pairs)
}

// nolint:unused
func testWatch(t *testing.T, client *EtcdV3) {
	prefix := "testWatch"
	key, value := "testWatch/node", "node"

	events := client.Watch(context.Background(), prefix)

	// update loop
	go func() {
		time.Sleep(500 * time.Millisecond)

		err := client.Create(context.Background(), key, value)
		assert.Nil(t, err)

		err = client.Delete(context.Background(), key)
		assert.Nil(t, err)
	}()

	eventCount := 0
	for {
		select {
		case event := <-events:
			assert.NotNil(t, event)
			assert.Nil(t, event.Error)
			assert.Len(t, event.Events, 1)

			vals, err := event.Decode()
			assert.Nil(t, err)
			val := vals[0]

			if eventCount == 0 {
				assert.Equal(t, utils.EventAdd, val[0])
			} else if eventCount == 1 {
				assert.Equal(t, utils.EventDelete, val[0])
			}

			assert.Equal(t, key, val[1])
			assert.Equal(t, val, val[2])

			eventCount += 1
			// We received all the events we wanted to check
			if eventCount == 2 {
				return
			}
		case <-time.After(5 * time.Second):
			assert.True(t, false, "Test watch timeout reached")
		}
	}
}
