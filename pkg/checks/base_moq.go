// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package checks

import (
	"context"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// Ensure, that CheckMock does implement Check.
// If this is not the case, regenerate this file with moq.
var _ Check = &CheckMock{}

// CheckMock is a mock implementation of Check.
//
//	func TestSomethingThatUsesCheck(t *testing.T) {
//
//		// make and configure a mocked Check
//		mockedCheck := &CheckMock{
//			GetConfigFunc: func() Runtime {
//				panic("mock out the GetConfig method")
//			},
//			GetMetricCollectorsFunc: func() []prometheus.Collector {
//				panic("mock out the GetMetricCollectors method")
//			},
//			NameFunc: func() string {
//				panic("mock out the Name method")
//			},
//			RemoveLabelledMetricsFunc: func(target string) error {
//				panic("mock out the RemoveLabelledMetrics method")
//			},
//			RunFunc: func(ctx context.Context, cResult chan ResultDTO) error {
//				panic("mock out the Run method")
//			},
//			SchemaFunc: func() (*openapi3.SchemaRef, error) {
//				panic("mock out the Schema method")
//			},
//			ShutdownFunc: func()  {
//				panic("mock out the Shutdown method")
//			},
//			UpdateConfigFunc: func(config Runtime) error {
//				panic("mock out the UpdateConfig method")
//			},
//		}
//
//		// use mockedCheck in code that requires Check
//		// and then make assertions.
//
//	}
type CheckMock struct {
	// GetConfigFunc mocks the GetConfig method.
	GetConfigFunc func() Runtime

	// GetMetricCollectorsFunc mocks the GetMetricCollectors method.
	GetMetricCollectorsFunc func() []prometheus.Collector

	// NameFunc mocks the Name method.
	NameFunc func() string

	// RemoveLabelledMetricsFunc mocks the RemoveLabelledMetrics method.
	RemoveLabelledMetricsFunc func(target string) error

	// RunFunc mocks the Run method.
	RunFunc func(ctx context.Context, cResult chan ResultDTO) error

	// SchemaFunc mocks the Schema method.
	SchemaFunc func() (*openapi3.SchemaRef, error)

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func()

	// UpdateConfigFunc mocks the UpdateConfig method.
	UpdateConfigFunc func(config Runtime) error

	// calls tracks calls to the methods.
	calls struct {
		// GetConfig holds details about calls to the GetConfig method.
		GetConfig []struct {
		}
		// GetMetricCollectors holds details about calls to the GetMetricCollectors method.
		GetMetricCollectors []struct {
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
		// RemoveLabelledMetrics holds details about calls to the RemoveLabelledMetrics method.
		RemoveLabelledMetrics []struct {
			// Target is the target argument value.
			Target string
		}
		// Run holds details about calls to the Run method.
		Run []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// CResult is the cResult argument value.
			CResult chan ResultDTO
		}
		// Schema holds details about calls to the Schema method.
		Schema []struct {
		}
		// Shutdown holds details about calls to the Shutdown method.
		Shutdown []struct {
		}
		// UpdateConfig holds details about calls to the UpdateConfig method.
		UpdateConfig []struct {
			// Config is the config argument value.
			Config Runtime
		}
	}
	lockGetConfig             sync.RWMutex
	lockGetMetricCollectors   sync.RWMutex
	lockName                  sync.RWMutex
	lockRemoveLabelledMetrics sync.RWMutex
	lockRun                   sync.RWMutex
	lockSchema                sync.RWMutex
	lockShutdown              sync.RWMutex
	lockUpdateConfig          sync.RWMutex
}

// GetConfig calls GetConfigFunc.
func (mock *CheckMock) GetConfig() Runtime {
	if mock.GetConfigFunc == nil {
		panic("CheckMock.GetConfigFunc: method is nil but Check.GetConfig was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetConfig.Lock()
	mock.calls.GetConfig = append(mock.calls.GetConfig, callInfo)
	mock.lockGetConfig.Unlock()
	return mock.GetConfigFunc()
}

// GetConfigCalls gets all the calls that were made to GetConfig.
// Check the length with:
//
//	len(mockedCheck.GetConfigCalls())
func (mock *CheckMock) GetConfigCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetConfig.RLock()
	calls = mock.calls.GetConfig
	mock.lockGetConfig.RUnlock()
	return calls
}

// GetMetricCollectors calls GetMetricCollectorsFunc.
func (mock *CheckMock) GetMetricCollectors() []prometheus.Collector {
	if mock.GetMetricCollectorsFunc == nil {
		panic("CheckMock.GetMetricCollectorsFunc: method is nil but Check.GetMetricCollectors was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetMetricCollectors.Lock()
	mock.calls.GetMetricCollectors = append(mock.calls.GetMetricCollectors, callInfo)
	mock.lockGetMetricCollectors.Unlock()
	return mock.GetMetricCollectorsFunc()
}

// GetMetricCollectorsCalls gets all the calls that were made to GetMetricCollectors.
// Check the length with:
//
//	len(mockedCheck.GetMetricCollectorsCalls())
func (mock *CheckMock) GetMetricCollectorsCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetMetricCollectors.RLock()
	calls = mock.calls.GetMetricCollectors
	mock.lockGetMetricCollectors.RUnlock()
	return calls
}

// Name calls NameFunc.
func (mock *CheckMock) Name() string {
	if mock.NameFunc == nil {
		panic("CheckMock.NameFunc: method is nil but Check.Name was just called")
	}
	callInfo := struct {
	}{}
	mock.lockName.Lock()
	mock.calls.Name = append(mock.calls.Name, callInfo)
	mock.lockName.Unlock()
	return mock.NameFunc()
}

// NameCalls gets all the calls that were made to Name.
// Check the length with:
//
//	len(mockedCheck.NameCalls())
func (mock *CheckMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// RemoveLabelledMetrics calls RemoveLabelledMetricsFunc.
func (mock *CheckMock) RemoveLabelledMetrics(target string) error {
	if mock.RemoveLabelledMetricsFunc == nil {
		panic("CheckMock.RemoveLabelledMetricsFunc: method is nil but Check.RemoveLabelledMetrics was just called")
	}
	callInfo := struct {
		Target string
	}{
		Target: target,
	}
	mock.lockRemoveLabelledMetrics.Lock()
	mock.calls.RemoveLabelledMetrics = append(mock.calls.RemoveLabelledMetrics, callInfo)
	mock.lockRemoveLabelledMetrics.Unlock()
	return mock.RemoveLabelledMetricsFunc(target)
}

// RemoveLabelledMetricsCalls gets all the calls that were made to RemoveLabelledMetrics.
// Check the length with:
//
//	len(mockedCheck.RemoveLabelledMetricsCalls())
func (mock *CheckMock) RemoveLabelledMetricsCalls() []struct {
	Target string
} {
	var calls []struct {
		Target string
	}
	mock.lockRemoveLabelledMetrics.RLock()
	calls = mock.calls.RemoveLabelledMetrics
	mock.lockRemoveLabelledMetrics.RUnlock()
	return calls
}

// Run calls RunFunc.
func (mock *CheckMock) Run(ctx context.Context, cResult chan ResultDTO) error {
	if mock.RunFunc == nil {
		panic("CheckMock.RunFunc: method is nil but Check.Run was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		CResult chan ResultDTO
	}{
		Ctx:     ctx,
		CResult: cResult,
	}
	mock.lockRun.Lock()
	mock.calls.Run = append(mock.calls.Run, callInfo)
	mock.lockRun.Unlock()
	return mock.RunFunc(ctx, cResult)
}

// RunCalls gets all the calls that were made to Run.
// Check the length with:
//
//	len(mockedCheck.RunCalls())
func (mock *CheckMock) RunCalls() []struct {
	Ctx     context.Context
	CResult chan ResultDTO
} {
	var calls []struct {
		Ctx     context.Context
		CResult chan ResultDTO
	}
	mock.lockRun.RLock()
	calls = mock.calls.Run
	mock.lockRun.RUnlock()
	return calls
}

// Schema calls SchemaFunc.
func (mock *CheckMock) Schema() (*openapi3.SchemaRef, error) {
	if mock.SchemaFunc == nil {
		panic("CheckMock.SchemaFunc: method is nil but Check.Schema was just called")
	}
	callInfo := struct {
	}{}
	mock.lockSchema.Lock()
	mock.calls.Schema = append(mock.calls.Schema, callInfo)
	mock.lockSchema.Unlock()
	return mock.SchemaFunc()
}

// SchemaCalls gets all the calls that were made to Schema.
// Check the length with:
//
//	len(mockedCheck.SchemaCalls())
func (mock *CheckMock) SchemaCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockSchema.RLock()
	calls = mock.calls.Schema
	mock.lockSchema.RUnlock()
	return calls
}

// Shutdown calls ShutdownFunc.
func (mock *CheckMock) Shutdown() {
	if mock.ShutdownFunc == nil {
		panic("CheckMock.ShutdownFunc: method is nil but Check.Shutdown was just called")
	}
	callInfo := struct {
	}{}
	mock.lockShutdown.Lock()
	mock.calls.Shutdown = append(mock.calls.Shutdown, callInfo)
	mock.lockShutdown.Unlock()
	mock.ShutdownFunc()
}

// ShutdownCalls gets all the calls that were made to Shutdown.
// Check the length with:
//
//	len(mockedCheck.ShutdownCalls())
func (mock *CheckMock) ShutdownCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}

// UpdateConfig calls UpdateConfigFunc.
func (mock *CheckMock) UpdateConfig(config Runtime) error {
	if mock.UpdateConfigFunc == nil {
		panic("CheckMock.UpdateConfigFunc: method is nil but Check.UpdateConfig was just called")
	}
	callInfo := struct {
		Config Runtime
	}{
		Config: config,
	}
	mock.lockUpdateConfig.Lock()
	mock.calls.UpdateConfig = append(mock.calls.UpdateConfig, callInfo)
	mock.lockUpdateConfig.Unlock()
	return mock.UpdateConfigFunc(config)
}

// UpdateConfigCalls gets all the calls that were made to UpdateConfig.
// Check the length with:
//
//	len(mockedCheck.UpdateConfigCalls())
func (mock *CheckMock) UpdateConfigCalls() []struct {
	Config Runtime
} {
	var calls []struct {
		Config Runtime
	}
	mock.lockUpdateConfig.RLock()
	calls = mock.calls.UpdateConfig
	mock.lockUpdateConfig.RUnlock()
	return calls
}
