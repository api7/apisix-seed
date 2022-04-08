package discoverer

import (
	"github.com/api7/apisix-seed/internal/core/message"
	"github.com/stretchr/testify/mock"
)

type MockInterface struct {
	mock.Mock
}

func NewDiscovererMock(_ interface{}) (Discoverer, error) {
	return &MockInterface{}, nil
}

func (m *MockInterface) Stop() {
	_ = m.Called()
}

func (m *MockInterface) Query(msg *message.Message) error {
	ret := m.Called(msg)
	return ret.Error(0)
}

func (m *MockInterface) Update(oldMsg, msg *message.Message) error {
	ret := m.Called(oldMsg, msg)
	return ret.Error(0)
}

func (m *MockInterface) Delete(msg *message.Message) error {
	ret := m.Called(msg)
	return ret.Error(0)
}

func (m *MockInterface) Watch() chan *message.Message {
	ret := m.Called()
	return ret.Get(0).(chan *message.Message)
}
