// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// Ensure, that ProviderMock does implement Provider.
// If this is not the case, regenerate this file with moq.
var _ Provider = &ProviderMock{}

// ProviderMock is a mock implementation of Provider.
//
//	func TestSomethingThatUsesProvider(t *testing.T) {
//
//		// make and configure a mocked Provider
//		mockedProvider := &ProviderMock{
//			GetRegistryFunc: func() *prometheus.Registry {
//				panic("mock out the GetRegistry method")
//			},
//			InitTracingFunc: func(ctx context.Context) error {
//				panic("mock out the InitTracing method")
//			},
//			ShutdownFunc: func(ctx context.Context) error {
//				panic("mock out the Shutdown method")
//			},
//		}
//
//		// use mockedProvider in code that requires Provider
//		// and then make assertions.
//
//	}
type ProviderMock struct {
	// GetRegistryFunc mocks the GetRegistry method.
	GetRegistryFunc func() *prometheus.Registry

	// InitTracingFunc mocks the InitTracing method.
	InitTracingFunc func(ctx context.Context) error

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func(ctx context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// GetRegistry holds details about calls to the GetRegistry method.
		GetRegistry []struct {
		}
		// InitTracing holds details about calls to the InitTracing method.
		InitTracing []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// Shutdown holds details about calls to the Shutdown method.
		Shutdown []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
	}
	lockGetRegistry sync.RWMutex
	lockInitTracing sync.RWMutex
	lockShutdown    sync.RWMutex
}

// GetRegistry calls GetRegistryFunc.
func (mock *ProviderMock) GetRegistry() *prometheus.Registry {
	if mock.GetRegistryFunc == nil {
		panic("ProviderMock.GetRegistryFunc: method is nil but Provider.GetRegistry was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetRegistry.Lock()
	mock.calls.GetRegistry = append(mock.calls.GetRegistry, callInfo)
	mock.lockGetRegistry.Unlock()
	return mock.GetRegistryFunc()
}

// GetRegistryCalls gets all the calls that were made to GetRegistry.
// Check the length with:
//
//	len(mockedProvider.GetRegistryCalls())
func (mock *ProviderMock) GetRegistryCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetRegistry.RLock()
	calls = mock.calls.GetRegistry
	mock.lockGetRegistry.RUnlock()
	return calls
}

// InitTracing calls InitTracingFunc.
func (mock *ProviderMock) InitTracing(ctx context.Context) error {
	if mock.InitTracingFunc == nil {
		panic("ProviderMock.InitTracingFunc: method is nil but Provider.InitTracing was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockInitTracing.Lock()
	mock.calls.InitTracing = append(mock.calls.InitTracing, callInfo)
	mock.lockInitTracing.Unlock()
	return mock.InitTracingFunc(ctx)
}

// InitTracingCalls gets all the calls that were made to InitTracing.
// Check the length with:
//
//	len(mockedProvider.InitTracingCalls())
func (mock *ProviderMock) InitTracingCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockInitTracing.RLock()
	calls = mock.calls.InitTracing
	mock.lockInitTracing.RUnlock()
	return calls
}

// Shutdown calls ShutdownFunc.
func (mock *ProviderMock) Shutdown(ctx context.Context) error {
	if mock.ShutdownFunc == nil {
		panic("ProviderMock.ShutdownFunc: method is nil but Provider.Shutdown was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockShutdown.Lock()
	mock.calls.Shutdown = append(mock.calls.Shutdown, callInfo)
	mock.lockShutdown.Unlock()
	return mock.ShutdownFunc(ctx)
}

// ShutdownCalls gets all the calls that were made to Shutdown.
// Check the length with:
//
//	len(mockedProvider.ShutdownCalls())
func (mock *ProviderMock) ShutdownCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}
