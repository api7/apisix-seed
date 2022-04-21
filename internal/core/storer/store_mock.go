package storer

import (
	"context"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/stretchr/testify/mock"
)

type MockInterface struct {
	mock.Mock
}

func (m *MockInterface) List(_ context.Context, key string) ([]*message.Message, error) {
	ret := m.Called(key)
	return ret.Get(0).([]*message.Message), ret.Error(1)
}

func (m *MockInterface) Update(_ context.Context, key, value string, version int64) error {
	ret := m.Called(key, value, version)
	return ret.Error(0)
}

func (m *MockInterface) Watch(_ context.Context, key string) <-chan []*message.Message {
	ret := m.Called(key)
	return ret.Get(0).(chan []*message.Message)
}
