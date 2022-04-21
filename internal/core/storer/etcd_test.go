package storer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/stretchr/testify/assert"
)

var host = "localhost:2379" // nolint:unused

func TestEtcdV3(t *testing.T) {
	// Make sure that etcd is installed in your test environment
	// Then comment out the following statement

	client, err := NewEtcd(&conf.Etcd{Host: []string{host}})
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
		err = client.Update(context.Background(), key, newValue, 1)
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
		wantErr := fmt.Errorf("etcd delete key[%s] is not found", key)
		assert.Equal(t, wantErr, err, "Test delete non-existing key")

		// Get should fail
		_, err = client.Get(context.Background(), key)
		wantErr = fmt.Errorf("etcd get key[%s] is not found", key)
		assert.Equal(t, wantErr, err, "Test get non-existing key")
	}
}

// nolint:unused
func testList(t *testing.T, client *EtcdV3) {
	prefix := "testList"
	kvs := map[string]string{
		"testList/first":  getA6Conf("first"),
		"testList/second": getA6Conf("second"),
		"testList/":       "init_dir",
	}

	for k, v := range kvs {
		err := client.Create(context.Background(), k, v)
		assert.Nil(t, err)
	}

	for _, parent := range []string{prefix, prefix + "/"} {
		pairs, err := client.List(context.Background(), parent)
		assert.Nil(t, err, "Test list prefix")
		assert.Len(t, pairs, 2, "Test list content")

		for _, pair := range pairs {
			if pair.Key == "dirPlaceholderKey" {
				assert.Fail(t, "should be skipped")
			}
			assert.Equal(t, kvs[pair.Key], pair.Value)
		}
	}

	err := client.DeletePrefix(context.Background(), prefix)
	assert.Nil(t, err, "Test delete prefix")

	// List should fail
	wantErr := fmt.Errorf("etcd list prefix[%s] is not found", prefix)
	pairs, err := client.List(context.Background(), prefix)
	assert.Equal(t, wantErr, err, "Test list non-existing prefix")
	assert.Nil(t, pairs)
}

// nolint:unused
func testWatch(t *testing.T, client *EtcdV3) {
	prefix := "testWatch"
	key, value := "testWatch/node", getA6Conf("node")
	dirPlaceholderKey, dirPlaceholderValue := "testWatch/", "init_dir"

	msgsCh := client.Watch(context.Background(), prefix)
	// update loop
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)

		err := client.Create(context.Background(), key, value)
		assert.Nil(t, err)

		err = client.Delete(context.Background(), key)
		assert.Nil(t, err)

		err = client.Create(context.Background(), dirPlaceholderKey, dirPlaceholderValue)
		assert.Nil(t, err)
	}()

	eventCount := 0
	addFlag, delFlag := false, false
	for {
		select {
		case msgs := <-msgsCh:
			eventCount += len(msgs)
			assert.True(t, eventCount < 3)
			for _, msg := range msgs {
				switch msg.Action {
				case message.EventAdd:
					addFlag = true
				case message.EventDelete:
					delFlag = true
				}
				assert.Equal(t, key, msg.Key)
			}
			if addFlag && delFlag {
				wg.Wait()
				return
			}
		case <-time.After(5 * time.Second):
			assert.True(t, false, "Test watch timeout reached")
			return
		}
	}

}

func getA6Conf(uri string) string {
	a6fmt := `{
		"uri": "%s",
			"upstream": {
                "discovery_type": "nacos",
				"service_name": "APISIX-NACOS",
				"discovery_args": {
				"group_name": "DEFAULT_GROUP"
			}
		}
	}`
	return fmt.Sprintf(a6fmt, uri)
}
