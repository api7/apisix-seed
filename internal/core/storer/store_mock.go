package storer

import (
	"context"

	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/mock"
)

type MockInterface struct {
	mock.Mock
}

func (m *MockInterface) List(_ context.Context, key string) (utils.Message, error) {
	ret := m.Called(key)
	return ret.Get(0).(utils.Message), ret.Error(1)
}

func (m *MockInterface) Update(_ context.Context, key, value string) error {
	ret := m.Called(key, value)
	return ret.Error(0)
}

func (m *MockInterface) Watch(_ context.Context, key string) <-chan *Watch {
	ret := m.Called(key)
	return ret.Get(0).(chan *Watch)
}
