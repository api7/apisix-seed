package discoverer

import (
	"github.com/api7/apisix-seed/internal/core/comm"
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

func (m *MockInterface) Query(query *comm.Query) error {
	ret := m.Called(query)
	return ret.Error(0)
}

func (m *MockInterface) Update(update *comm.Update) error {
	ret := m.Called(update)
	return ret.Error(0)
}

func (m *MockInterface) Watch() chan *comm.Message {
	ret := m.Called()
	return ret.Get(0).(chan *comm.Message)
}
