package storer

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/conf"

	"github.com/api7/apisix-seed/internal/log"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	DirPlaceholder = []byte("init_dir")
)

type EtcdV3 struct {
	client  *clientv3.Client
	conf    clientv3.Config
	timeout time.Duration
}

func NewEtcd(etcdConf *conf.Etcd) (*EtcdV3, error) {
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
		log.Errorf("etcd init failed: %s", err)
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
		log.Errorf("etcd get key[%s] failed: %s", key, err)
		return "", fmt.Errorf("etcd get key[%s] failed: %s", key, err)
	}
	if resp.Count == 0 {
		log.Warnf("etcd get key[%s] is not found", key)
		return "", fmt.Errorf("etcd get key[%s] is not found", key)
	}

	return string(resp.Kvs[0].Value), nil
}

// List the content of a given prefix
func (s *EtcdV3) List(ctx context.Context, prefix string) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Errorf("etcd list prefix[%s] failed: %s", prefix, err)
		return nil, fmt.Errorf("etcd list prefix[%s] failed: %s", prefix, err)
	}
	if resp.Count == 0 {
		log.Warnf("etcd list prefix[%s] is not found", prefix)
		return nil, fmt.Errorf("etcd list prefix[%s] is not found", prefix)
	}

	// We use a placeholder to mark a key to be a directory. So we need to skip the hack here.
	if bytes.Equal(resp.Kvs[0].Value, DirPlaceholder) {
		resp.Kvs = resp.Kvs[1:]
	}

	msgs := make([]*message.Message, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		msg, err := message.NewMessage(string(kv.Key), kv.Value, kv.Version, message.EventAdd)
		if err != nil {
			log.Errorf("etcd list prefix[%s] format failed: %s", prefix, err)
			continue
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

// Create a value at the specified key
func (s *EtcdV3) Create(ctx context.Context, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.Put(ctx, key, value)
	if err != nil {
		log.Errorf("etcd put key[%s] failed: %s", key, err)
		return fmt.Errorf("etcd put key[%s] failed: %s", key, err)
	}

	return nil
}

// Update a value at the specified key
func (s *EtcdV3) Update(ctx context.Context, key, value string, version int64) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	txn := s.client.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(key), "=", version)).
		Then(clientv3.OpPut(key, value))

	resp, err := txn.Commit()
	if err != nil {
		log.Errorf("etcd update key[%s] failed: %s", key, err)
		return fmt.Errorf("etcd update key[%s] failed: %s", key, err)
	}
	if !resp.Succeeded {
		log.Infof("key[%s] may have been updated by other instances", key)
	}
	log.Infof("etcd update key[%s], version: %d", key, version)
	return nil
}

// Delete a value at the specified key
func (s *EtcdV3) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Delete(ctx, key)
	if err != nil {
		log.Errorf("etcd delete key[%s] failed: %s", key, err)
		return fmt.Errorf("etcd delete key[%s] failed: %s", key, err)
	}
	if resp.Deleted == 0 {
		log.Warnf("etcd delete key[%s] is not found", key)
		return fmt.Errorf("etcd delete key[%s] is not found", key)
	}

	return nil
}

// DeletePrefix deletes a range of keys under a given prefix
func (s *EtcdV3) DeletePrefix(ctx context.Context, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Errorf("etcd delete prefix[%s] failed: %s", prefix, err)
		return fmt.Errorf("etcd delete prefix[%s] failed: %s", prefix, err)
	}
	if resp.Deleted == 0 {
		log.Warnf("etcd delete prefix[%s] is not found", prefix)
		return fmt.Errorf("etcd delete prefix[%s] is not found", prefix)
	}

	return nil
}

// Watch for changes on a key
func (s *EtcdV3) Watch(ctx context.Context, prefix string) <-chan []*message.Message {
	eventChan := s.client.Watch(ctx, prefix, clientv3.WithPrefix())
	ch := make(chan []*message.Message, 1)

	go func() {
		defer close(ch)

		for event := range eventChan {
			msgs := make([]*message.Message, 0, 16)

			for _, ev := range event.Events {
				// We use a placeholder to mark a key to be a directory. So we need to skip the hack here.
				if bytes.Equal(ev.Kv.Value, DirPlaceholder) {
					continue
				}

				key := string(ev.Kv.Key)

				var typ message.StoreEvent
				switch ev.Type {
				case clientv3.EventTypePut:
					typ = message.EventAdd
				case clientv3.EventTypeDelete:
					typ = message.EventDelete
				}

				log.Infof("watch changed, key: %s, version: %d", key, ev.Kv.Version)
				msg, err := message.NewMessage(key, ev.Kv.Value, ev.Kv.Version, typ)
				if err != nil {
					log.Warnf("etcd watch key[%s]'s %s event failed: %s", key, typ, err)
					continue
				}
				msgs = append(msgs, msg)
			}

			ch <- msgs
		}
	}()

	return ch
}

// Close the client connection
func (s *EtcdV3) Close() error {
	if err := s.client.Close(); err != nil {
		log.Errorf("etcd client close failed: %s", err)
		return err
	}
	return nil
}
