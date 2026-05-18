package vm

import "context"

type MockClient struct {
	CreateSandboxFn func(ctx context.Context, id string, specYAML string) (*Sandbox, error)
	GetSandboxFn    func(ctx context.Context, id string) (*Sandbox, error)
	DeleteSandboxFn func(ctx context.Context, id string) error
}

func (m *MockClient) CreateSandbox(ctx context.Context, id string, specYAML string) (*Sandbox, error) {
	if m.CreateSandboxFn == nil {
		return nil, ErrUnexpected
	}

	return m.CreateSandboxFn(ctx, id, specYAML)
}

func (m *MockClient) GetSandbox(ctx context.Context, id string) (*Sandbox, error) {
	if m.GetSandboxFn == nil {
		return nil, ErrUnexpected
	}

	return m.GetSandboxFn(ctx, id)
}

func (m *MockClient) DeleteSandbox(ctx context.Context, id string) error {
	if m.DeleteSandboxFn == nil {
		return ErrUnexpected
	}

	return m.DeleteSandboxFn(ctx, id)
}
