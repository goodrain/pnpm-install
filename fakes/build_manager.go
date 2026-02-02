package fakes

import (
	"sync"

	npminstall "github.com/goodrain/pnpm-install"
)

type BuildManager struct {
	ResolveCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			BuildProcess pnpminstall.BuildProcess
			Bool         bool
			Error        error
		}
		Stub func(string) (pnpminstall.BuildProcess, bool, error)
	}
}

func (f *BuildManager) Resolve(param1 string) (pnpminstall.BuildProcess, bool, error) {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.WorkingDir = param1
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1)
	}
	return f.ResolveCall.Returns.BuildProcess, f.ResolveCall.Returns.Bool, f.ResolveCall.Returns.Error
}
