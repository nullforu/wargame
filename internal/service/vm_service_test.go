package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/config"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/utils"
	"wargame/internal/vm"

	"golang.org/x/crypto/bcrypt"
)

const testVMSpec = `apiVersion: sandboxd.o/v1
kind: Sandbox
id: placeholder
spec:
  egress: true
  ttl_seconds: 3600
  ports:
    - host_port: 0
      container_port: 31337
      protocol: tcp
  containers:
    - name: app
      image: nginx:latest
      resource:
        cpu: 50m
        memory: 64Mi
`

func createVMChallenge(t *testing.T, env serviceEnv, title string) *models.Challenge {
	t.Helper()
	spec := testVMSpec
	challenge := &models.Challenge{
		Title:       title,
		Description: "desc",
		Category:    "Web",
		Points:      100,
		VMEnabled:   true,
		VMSpec:      &spec,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
	}

	hash, err := utils.HashFlag("flag", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}
	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func newVMServiceForTest(env serviceEnv, client vm.API, cfg config.VMConfig) (*VMService, *repo.VMRepo) {
	vmRepo := repo.NewVMRepo(env.db)
	return NewVMService(cfg, vmRepo, env.challengeRepo, env.submissionRepo, client, env.redis), vmRepo
}

func TestVMServiceGetOrCreateVMRewritesManifestID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-user@example.com", "vm-user", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm")
	var requestedID string

	client := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			requestedID = id
			_, req, err := vm.RenderManifestWithID(specYAML, id)
			if err != nil {
				t.Fatalf("render manifest: %v", err)
			}

			if req.ID != id {
				t.Fatalf("manifest id not rewritten: got %q want %q", req.ID, id)
			}

			exp := time.Now().UTC().Add(time.Hour)
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Pending", ExpireAt: &exp}}, nil
		},
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Running", ExternalIP: "127.0.0.1", AssignedPorts: []vm.PortMapping{{HostPort: 31000, ContainerPort: 31337, Protocol: "tcp"}}}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, id string) error { return nil },
	}
	svc, _ := newVMServiceForTest(env, client, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	model, err := svc.GetOrCreateVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateVM: %v", err)
	}

	if model.VMID == "" || requestedID != model.VMID {
		t.Fatalf("unexpected vm id: requested=%q model=%q", requestedID, model.VMID)
	}
}

func TestVMServiceGetOrCreateVMAllowsSolvedChallenge(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-solved@example.com", "vm-solved", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-solved")
	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	client := &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			exp := time.Now().UTC().Add(time.Hour)
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Pending", ExpireAt: &exp}}, nil
		},
	}
	svc, _ := newVMServiceForTest(env, client, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	model, err := svc.GetOrCreateVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateVM solved challenge: %v", err)
	}

	if model == nil || model.VMID == "" {
		t.Fatalf("expected created vm for solved challenge, got %+v", model)
	}
}

func TestVMServiceRefreshDoesNotDeleteFailedVM(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-failed@example.com", "vm-failed", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-failed")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Failed", LastError: "image pull failed"}}, nil
		},
		DeleteSandboxFn: func(ctx context.Context, id string) error { return nil },
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-failed-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	got, err := svc.GetVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetVM: %v", err)
	}

	if got.Status != "Failed" || got.LastError == nil || *got.LastError != "image pull failed" {
		t.Fatalf("expected failed vm with error, got %+v", got)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-failed-1"); err != nil {
		t.Fatalf("vm should remain in db: %v", err)
	}
}

func TestVMServiceRefreshReturnsUnavailableWithoutLastError(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-unavailable@example.com", "vm-unavailable", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-unavailable")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, vm.ErrUnavailable
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-unavailable-1", Status: "Pending", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	if _, err := svc.GetVM(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrVMOrchestratorDown) {
		t.Fatalf("expected ErrVMOrchestratorDown, got %v", err)
	}

	got, err := vmRepo.GetByVMID(context.Background(), "vm-unavailable-1")
	if err != nil {
		t.Fatalf("vm should remain in db: %v", err)
	}

	if got.LastError != nil {
		t.Fatalf("network errors should not be stored as last error: %q", *got.LastError)
	}
}

func TestVMServiceRefreshStoresOrchestratorHTTPError(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-http-error@example.com", "vm-http-error", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-http-error")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 409, Message: "sandbox is terminating"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-http-error-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	got, err := svc.GetVM(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetVM should keep HTTP response errors in last_error: %v", err)
	}

	if got.LastError == nil || *got.LastError != "sandbox is terminating" {
		t.Fatalf("expected orchestrator message in last_error, got %+v", got.LastError)
	}

	if got.Status != "Error" {
		t.Fatalf("expected status Error on orchestrator HTTP error, got %q", got.Status)
	}
}

func TestVMServiceRefreshDeletesMissingVM(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-missing@example.com", "vm-missing", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-missing")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 404, Message: "not found"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-missing-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	if _, err := svc.GetVM(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrVMNotFound) {
		t.Fatalf("expected ErrVMNotFound, got %v", err)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-missing-1"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected vm row to be deleted, got %v", err)
	}
}

func TestVMServiceAdminListDoesNotRefreshRows(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-admin-refresh@example.com", "vm-admin-refresh", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-admin-refresh")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return nil, &vm.StatusError{StatusCode: 404, Message: "not found"}
		},
	}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{UserID: user.ID, ChallengeID: challenge.ID, VMID: "vm-admin-refresh-1", Status: "Running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	rows, err := svc.ListAdminVMs(context.Background())
	if err != nil {
		t.Fatalf("ListAdminVMs: %v", err)
	}

	if len(rows) != 1 || rows[0].VMID != "vm-admin-refresh-1" {
		t.Fatalf("expected vm row to remain in list without refresh, got %+v", rows)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-admin-refresh-1"); err != nil {
		t.Fatalf("expected vm row to remain in db without refresh, got %v", err)
	}
}

func TestVMServiceGetOrCreateVMCleansUpExpiredRowsBeforeLimit(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-cleanup-limit@example.com", "vm-cleanup-limit", "pass", models.UserRole)
	challenge1 := createVMChallenge(t, env, "vm-cleanup-limit-1")
	challenge2 := createVMChallenge(t, env, "vm-cleanup-limit-2")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			exp := time.Now().UTC().Add(time.Hour)
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Pending", ExpireAt: &exp}}, nil
		},
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Running"}}, nil
		},
	}, config.VMConfig{Enabled: true, MaxPer: 1, CreateWindow: time.Minute, CreateMax: 5, CleanupInterval: 30 * time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:       user.ID,
		ChallengeID:  challenge1.ID,
		VMID:         "vm-expired-limit",
		Status:       "Running",
		TTLExpiresAt: ptrTime(now.Add(-time.Minute)),
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create expired vm: %v", err)
	}

	created, err := svc.GetOrCreateVM(context.Background(), user.ID, challenge2.ID)
	if err != nil {
		t.Fatalf("GetOrCreateVM: %v", err)
	}
	if created == nil || created.VMID == "" {
		t.Fatalf("expected created vm, got %+v", created)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-expired-limit"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected expired vm to be removed before limit check, got %v", err)
	}
}

func TestVMServiceGetOrCreateVMKeepsRowsWithNullTTL(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-null-ttl@example.com", "vm-null-ttl", "pass", models.UserRole)
	challenge1 := createVMChallenge(t, env, "vm-null-ttl-1")
	challenge2 := createVMChallenge(t, env, "vm-null-ttl-2")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		CreateSandboxFn: func(ctx context.Context, id string, specYAML string) (*vm.Sandbox, error) {
			exp := time.Now().UTC().Add(time.Hour)
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Pending", ExpireAt: &exp}}, nil
		},
	}, config.VMConfig{Enabled: true, MaxPer: 1, CreateWindow: time.Minute, CreateMax: 5, CleanupInterval: 30 * time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge1.ID,
		VMID:        "vm-null-ttl-limit",
		Status:      "Running",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create null ttl vm: %v", err)
	}

	if _, err := svc.GetOrCreateVM(context.Background(), user.ID, challenge2.ID); !errors.Is(err, ErrVMLimitReached) {
		t.Fatalf("expected ErrVMLimitReached because null ttl row remains, got %v", err)
	}
}

func TestVMServiceCleanupExpiredVMs(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-cleanup-all@example.com", "vm-cleanup-all", "pass", models.UserRole)
	challenge1 := createVMChallenge(t, env, "vm-cleanup-all-1")
	challenge2 := createVMChallenge(t, env, "vm-cleanup-all-2")
	challenge3 := createVMChallenge(t, env, "vm-cleanup-all-3")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 5, CleanupInterval: 30 * time.Minute})

	now := time.Now().UTC()
	for _, row := range []*models.VM{
		{UserID: user.ID, ChallengeID: challenge1.ID, VMID: "vm-clean-expired", Status: "Running", TTLExpiresAt: ptrTime(now.Add(-time.Minute)), CreatedAt: now, UpdatedAt: now},
		{UserID: user.ID, ChallengeID: challenge2.ID, VMID: "vm-clean-active", Status: "Running", TTLExpiresAt: ptrTime(now.Add(time.Minute)), CreatedAt: now, UpdatedAt: now},
		{UserID: user.ID, ChallengeID: challenge3.ID, VMID: "vm-clean-null", Status: "Running", CreatedAt: now, UpdatedAt: now},
	} {
		if err := vmRepo.Create(context.Background(), row); err != nil {
			t.Fatalf("create vm(%s): %v", row.VMID, err)
		}
	}

	deleted, err := svc.CleanupExpiredVMs(context.Background())
	if err != nil {
		t.Fatalf("CleanupExpiredVMs: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected deleted=1, got %d", deleted)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-clean-expired"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected expired vm deleted, got %v", err)
	}
	if _, err := vmRepo.GetByVMID(context.Background(), "vm-clean-active"); err != nil {
		t.Fatalf("expected active vm remain, got %v", err)
	}
	if _, err := vmRepo.GetByVMID(context.Background(), "vm-clean-null"); err != nil {
		t.Fatalf("expected null-ttl vm remain, got %v", err)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestVMServiceUserVMSummary(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-summary@example.com", "vm-summary", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-summary")
	_, vmRepo := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-summary-1",
		Status:      "Running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	svcEnabled, _ := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})
	count, limit, err := svcEnabled.UserVMSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserVMSummary(enabled): %v", err)
	}

	if count != 1 || limit != 2 {
		t.Fatalf("expected 1/2, got %d/%d", count, limit)
	}

	svcDisabled, _ := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: false, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})
	count, limit, err = svcDisabled.UserVMSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserVMSummary(disabled): %v", err)
	}

	if count != 0 || limit != 0 {
		t.Fatalf("expected 0/0 when disabled, got %d/%d", count, limit)
	}
}

func TestVMServiceListAndGetByVMID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-list-get@example.com", "vm-list-get", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-list-get")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		GetSandboxFn: func(ctx context.Context, id string) (*vm.Sandbox, error) {
			return &vm.Sandbox{ID: id, Status: vm.SandboxStatus{Phase: "Running", ExternalIP: "127.0.0.1"}}, nil
		},
	}, config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-list-get-1",
		Status:      "Pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	rows, err := svc.ListUserVMs(context.Background(), user.ID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListUserVMs err=%v len=%d", err, len(rows))
	}

	row, err := svc.GetVMByVMID(context.Background(), "vm-list-get-1")
	if err != nil {
		t.Fatalf("GetVMByVMID: %v", err)
	}

	if row.Status != "Running" {
		t.Fatalf("expected refreshed status Running, got %s", row.Status)
	}
}

func TestVMServiceDeletePaths(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-delete@example.com", "vm-delete", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-delete")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{
		DeleteSandboxFn: func(ctx context.Context, id string) error { return vm.ErrNotFound },
	}, config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-delete-1",
		Status:      "Running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	if err := svc.DeleteVM(context.Background(), user.ID, challenge.ID); err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-delete-1"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected deleted row, got %v", err)
	}

	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		VMID:        "vm-delete-2",
		Status:      "Running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create vm2: %v", err)
	}

	if err := svc.DeleteVMByVMID(context.Background(), "vm-delete-2"); err != nil {
		t.Fatalf("DeleteVMByVMID: %v", err)
	}
}

func TestVMServiceStartTTLReaper(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-reaper@example.com", "vm-reaper", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-reaper")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:       user.ID,
		ChallengeID:  challenge.ID,
		VMID:         "vm-reaper-expired",
		Status:       "Running",
		TTLExpiresAt: ptrTime(now.Add(-time.Minute)),
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartTTLReaper(ctx, 10*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := vmRepo.GetByVMID(context.Background(), "vm-reaper-expired"); errors.Is(err, repo.ErrNotFound) {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("expected StartTTLReaper to delete expired vm row")
}

func TestVMServiceStartTTLReaperNonPositiveInterval(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vm-reaper-nonpos@example.com", "vm-reaper-nonpos", "pass", models.UserRole)
	challenge := createVMChallenge(t, env, "vm-reaper-nonpos")
	svc, vmRepo := newVMServiceForTest(env, &vm.MockClient{}, config.VMConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 1, CleanupInterval: time.Minute})

	now := time.Now().UTC()
	if err := vmRepo.Create(context.Background(), &models.VM{
		UserID:       user.ID,
		ChallengeID:  challenge.ID,
		VMID:         "vm-reaper-still-there",
		Status:       "Running",
		TTLExpiresAt: ptrTime(now.Add(-time.Minute)),
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("create vm: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartTTLReaper(ctx, 0)
	time.Sleep(50 * time.Millisecond)

	if _, err := vmRepo.GetByVMID(context.Background(), "vm-reaper-still-there"); err != nil {
		t.Fatalf("expected vm row to remain with non-positive interval, got %v", err)
	}
}

func TestVMServiceHelpers(t *testing.T) {
	if got := mapVMOrchestratorError(vm.ErrNotFound); !errors.Is(got, ErrVMNotFound) {
		t.Fatalf("expected ErrVMNotFound, got %v", got)
	}

	if got := mapVMOrchestratorError(vm.ErrInvalid); !errors.Is(got, ErrVMInvalidSpec) {
		t.Fatalf("expected ErrVMInvalidSpec, got %v", got)
	}

	if got := mapVMOrchestratorError(vm.ErrUnavailable); !errors.Is(got, ErrVMOrchestratorDown) {
		t.Fatalf("expected ErrVMOrchestratorDown, got %v", got)
	}

	ports := toVMPortMappings([]vm.PortMapping{{HostPort: 10000, ContainerPort: 31337, Protocol: "tcp"}})
	if len(ports) != 1 || ports[0].HostPort != 10000 {
		t.Fatalf("unexpected mapped ports: %+v", ports)
	}

	if toVMPortMappings(nil) != nil {
		t.Fatal("expected nil for empty port mappings")
	}

	if !isUniqueVMConflict(errors.New("duplicate key value violates unique constraint")) {
		t.Fatal("expected unique conflict=true")
	}

	if isUniqueVMConflict(errors.New("other error")) {
		t.Fatal("expected unique conflict=false")
	}
}
