package storer

import (
	"context"
	"fmt"
	"time"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/utils"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdV3 struct {
	client  *clientv3.Client
	conf    clientv3.Config
	timeout time.Duration
}

func NewEtcd(etcdConf conf.Etcd) (*EtcdV3, error) {
	timeout := time.Duration(etcdConf.Timeout)
	s := &EtcdV3{timeout: timeout}

	if s.timeout == 0 {
		s.timeout = 10 * time.Second
	}

	config := clientv3.Config{
		Endpoints:            etcdConf.Host,
		DialTimeout:          timeout,
		DialKeepAliveTimeout: timeout,
		Username:             etcdConf.User,
		Password:             etcdConf.Password,
	}

	if etcdConf.TLS != nil && etcdConf.TLS.Verify {
		tlsInfo := transport.TLSInfo{
			CertFile: etcdConf.TLS.CertFile,
			KeyFile:  etcdConf.TLS.KeyFile,
		}
		tlsConf, err := tlsInfo.ClientConfig()
		if err != nil {
			return nil, err
		}
		config.TLS = tlsConf
	}

	s.conf = config
	if err := s.init(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *EtcdV3) init() error {
	cli, err := clientv3.New(s.conf)
	if err != nil {
		return err
	}

	s.client = cli
	return nil
}

// Get a value given its key
func (s *EtcdV3) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("etcd get failed: %s", err)
	}
	if resp.Count == 0 {
		return "", fmt.Errorf("key: %s is not found", key)
	}

	return string(resp.Kvs[0].Value), nil
}

// List the content of a given prefix
func (s *EtcdV3) List(ctx context.Context, prefix string) (utils.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd list failed: %s", err)
	}
	if resp.Count == 0 {
		return nil, fmt.Errorf("prefix: %s is not found", prefix)
	}

	ret := make(utils.Message, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		ret.Add(string(kv.Key), string(kv.Value))
	}

	return ret, nil
}

// Create a value at the specified key
func (s *EtcdV3) Create(ctx context.Context, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("etcd put failed: %s", err)
	}

	return nil
}

// Update a value at the specified key
func (s *EtcdV3) Update(ctx context.Context, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("etcd put failed: %s", err)
	}

	return nil
}

// Delete a value at the specified key
func (s *EtcdV3) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete etcd key[%s] failed: %s", key, err)
	}
	if resp.Deleted == 0 {
		return fmt.Errorf("key: %s is not found", key)
	}

	return nil
}

// DeletePrefix deletes a range of keys under a given prefix
func (s *EtcdV3) DeletePrefix(ctx context.Context, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("delete etcd prefix[%s] failed: %s", prefix, err)
	}
	if resp.Deleted == 0 {
		return fmt.Errorf("prefix: %s is not found", prefix)
	}

	return nil
}

// Watch for changes on a key
func (s *EtcdV3) Watch(ctx context.Context, key string) <-chan *Watch {
	eventChan := s.client.Watch(ctx, key, clientv3.WithPrefix())
	ch := make(chan *Watch, 1)

	go func() {
		defer close(ch)

		for event := range eventChan {
			output := NewWatch(event.Canceled)

			for _, ev := range event.Events {
				key := string(ev.Kv.Key)
				value := string(ev.Kv.Value)

				var typ string
				switch ev.Type {
				case clientv3.EventTypePut:
					typ = utils.EventAdd
				case clientv3.EventTypeDelete:
					typ = utils.EventDelete
				}

				if err := output.Add(typ, key, value); err != nil {
					continue
				}
			}

			ch <- &output
		}
	}()

	return ch
}

// Close the client connection
func (s *EtcdV3) Close() error {
	if err := s.client.Close(); err != nil {
		return err
	}
	return nil
}
