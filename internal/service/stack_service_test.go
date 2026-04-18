package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/config"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/stack"
	"wargame/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

func createStackChallenge(t *testing.T, env serviceEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := &models.Challenge{
		Title:         title,
		Description:   "desc",
		Category:      "Web",
		Points:        100,
		MinimumPoints: 100,
		StackEnabled:  true,
		StackTargetPorts: stack.TargetPortSpecs{
			{ContainerPort: 80, Protocol: "TCP"},
		},
		StackPodSpec: &podSpec,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
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

func newStackService(env serviceEnv, client stack.API, cfg config.StackConfig) (*StackService, *repo.StackRepo) {
	stackRepo := repo.NewStackRepo(env.db)
	return NewStackService(cfg, stackRepo, env.challengeRepo, env.submissionRepo, client, env.redis), stackRepo
}

func TestStackServiceGetOrCreateStack(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "stacker@example.com", "stacker", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack")
	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack: %v", err)
	}
	if stackModel.StackID == "" || len(stackModel.Ports) != 1 {
		t.Fatalf("unexpected stack model: %+v", stackModel)
	}

	again, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack again: %v", err)
	}
	if again.StackID != stackModel.StackID || mock.CreateCount() != 1 {
		t.Fatalf("expected cached stack, calls=%d", mock.CreateCount())
	}
}

func TestStackServiceUserStackSummary(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "stack-summary@example.com", "stack-summary", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack-summary")
	terminalChallenge := createStackChallenge(t, env, "stack-summary-term")

	cfg := config.StackConfig{Enabled: true, MaxPer: 3, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, stack.NewProvisionerMock().Client(), cfg)

	now := time.Now().UTC()
	if err := stackRepo.Create(context.Background(), &models.Stack{UserID: user.ID, ChallengeID: challenge.ID, StackID: "stack-running", Status: "running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create running stack: %v", err)
	}
	if err := stackRepo.Create(context.Background(), &models.Stack{UserID: user.ID, ChallengeID: terminalChallenge.ID, StackID: "stack-stopped", Status: "stopped", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create stopped stack: %v", err)
	}

	count, limit, err := stackSvc.UserStackSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserStackSummary: %v", err)
	}
	if count != 1 || limit != cfg.MaxPer {
		t.Fatalf("expected 1/%d, got %d/%d", cfg.MaxPer, count, limit)
	}
}

func TestStackServiceListAndDeleteByStackID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "list-admin@example.com", "listadmin", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack-admin")
	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, stack.NewProvisionerMock().Client(), cfg)

	now := time.Now().UTC()
	if err := stackRepo.Create(context.Background(), &models.Stack{UserID: user.ID, ChallengeID: challenge.ID, StackID: "stack-admin-1", Status: "running", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	adminRows, err := stackSvc.ListAdminStacks(context.Background())
	if err != nil {
		t.Fatalf("ListAdminStacks: %v", err)
	}
	if len(adminRows) != 1 {
		t.Fatalf("expected 1 admin row, got %d", len(adminRows))
	}

	if err := stackSvc.DeleteStackByStackID(context.Background(), "stack-admin-1"); err != nil {
		t.Fatalf("DeleteStackByStackID: %v", err)
	}
	if _, err := stackSvc.GetStackByStackID(context.Background(), "stack-admin-1"); err == nil {
		t.Fatalf("expected missing stack after delete")
	}
}

func TestStackServiceDisabledAndNotFoundPaths(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "missing@example.com", "missing", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "missing")

	disabledSvc, _ := newStackService(env, stack.NewProvisionerMock().Client(), config.StackConfig{Enabled: false})
	if _, _, err := disabledSvc.UserStackSummary(context.Background(), user.ID); err != nil {
		t.Fatalf("UserStackSummary disabled should not error: %v", err)
	}
	if _, err := disabledSvc.GetStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackDisabled) {
		t.Fatalf("expected ErrStackDisabled, got %v", err)
	}
	if _, err := disabledSvc.ListAdminStacks(context.Background()); !errors.Is(err, ErrStackDisabled) {
		t.Fatalf("expected ErrStackDisabled, got %v", err)
	}

	enabledSvc, _ := newStackService(env, stack.NewProvisionerMock().Client(), config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})
	if _, err := enabledSvc.GetStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
	if err := enabledSvc.DeleteStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
	if err := enabledSvc.DeleteStackByUserAndChallenge(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
	if err := enabledSvc.DeleteStackByStackID(context.Background(), "missing"); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
	if _, err := enabledSvc.GetStackByStackID(context.Background(), "missing"); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
}

func TestStackServiceRateLimitAndUserLimit(t *testing.T) {
	rateEnv := setupServiceTest(t)
	user := createUser(t, rateEnv, "ratelimit@example.com", "ratelimit", "pass", models.UserRole)
	challenge1 := createStackChallenge(t, rateEnv, "rate-1")
	challenge2 := createStackChallenge(t, rateEnv, "rate-2")

	mock := stack.NewProvisionerMock()

	rateCfg := config.StackConfig{Enabled: true, MaxPer: 5, CreateWindow: time.Minute, CreateMax: 1}
	rateSvc, _ := newStackService(rateEnv, mock.Client(), rateCfg)
	if _, err := rateSvc.GetOrCreateStack(context.Background(), user.ID, challenge1.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := rateSvc.GetOrCreateStack(context.Background(), user.ID, challenge2.ID); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}

	limitEnv := setupServiceTest(t)
	limitUser := createUser(t, limitEnv, "limit@example.com", "limit", "pass", models.UserRole)
	limitChallenge1 := createStackChallenge(t, limitEnv, "limit-1")
	limitChallenge2 := createStackChallenge(t, limitEnv, "limit-2")

	limitMock := stack.NewProvisionerMock()
	limitCfg := config.StackConfig{Enabled: true, MaxPer: 1, CreateWindow: time.Minute, CreateMax: 5}
	limitSvc, _ := newStackService(limitEnv, limitMock.Client(), limitCfg)
	if _, err := limitSvc.GetOrCreateStack(context.Background(), limitUser.ID, limitChallenge1.ID); err != nil {
		t.Fatalf("first create(limit): %v", err)
	}
	if _, err := limitSvc.GetOrCreateStack(context.Background(), limitUser.ID, limitChallenge2.ID); !errors.Is(err, ErrStackLimitReached) {
		t.Fatalf("expected ErrStackLimitReached, got %v", err)
	}
}

func TestStackServiceChallengeStateAndSolvedPaths(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "state@example.com", "state", "pass", models.UserRole)

	notEnabled := createChallenge(t, env, "NoStack", 100, "FLAG{NS}", true)
	stackSvc, _ := newStackService(env, stack.NewProvisionerMock().Client(), config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, notEnabled.ID); !errors.Is(err, ErrStackNotEnabled) {
		t.Fatalf("expected ErrStackNotEnabled, got %v", err)
	}

	invalidSpec := createStackChallenge(t, env, "InvalidSpec")
	invalidSpec.StackTargetPorts = stack.TargetPortSpecs{{ContainerPort: 0, Protocol: "TCP"}}
	if err := env.challengeRepo.Update(context.Background(), invalidSpec); err != nil {
		t.Fatalf("update invalid challenge: %v", err)
	}
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, invalidSpec.ID); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec, got %v", err)
	}

	prev := createChallenge(t, env, "Prev", 50, "FLAG{PREV}", true)
	locked := createStackChallenge(t, env, "Locked")
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, locked.ID); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("expected ErrChallengeLocked, got %v", err)
	}
	createSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, locked.ID); err != nil {
		t.Fatalf("expected unlocked create, got %v", err)
	}

	solvedChallenge := createStackChallenge(t, env, "Solved")
	createSubmission(t, env, user.ID, solvedChallenge.ID, true, time.Now().UTC())
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, solvedChallenge.ID); !errors.Is(err, ErrAlreadySolved) {
		t.Fatalf("expected ErrAlreadySolved, got %v", err)
	}
}

func TestStackServiceProvisionerErrorMappingAndTerminalCleanup(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "prov@example.com", "prov", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "Prov")

	downClient := &stack.MockClient{CreateStackFn: func(ctx context.Context, targetPorts []stack.TargetPortSpec, podSpec string) (*stack.StackInfo, error) {
		return nil, stack.ErrUnavailable
	}}
	downSvc, _ := newStackService(env, downClient, config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})
	if _, err := downSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackProvisionerDown) {
		t.Fatalf("expected ErrStackProvisionerDown, got %v", err)
	}

	mock := stack.NewProvisionerMock()
	svc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})
	now := time.Now().UTC()
	stackModel := &models.Stack{UserID: user.ID, ChallengeID: challenge.ID, StackID: "terminal", Status: "running", CreatedAt: now, UpdatedAt: now}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}
	mock.SetStatus("terminal", "stopped")

	if _, err := svc.GetStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound for terminal stack, got %v", err)
	}
}

func TestStackServicePortSpecValidationHelpers(t *testing.T) {
	if _, err := toTargetPortSpecs(nil); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for empty ports, got %v", err)
	}
	if _, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 70000, Protocol: "TCP"}}); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for invalid port, got %v", err)
	}
	if _, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "ICMP"}}); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for invalid protocol, got %v", err)
	}

	ports, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "tcp"}})
	if err != nil || len(ports) != 1 || ports[0].Protocol != "TCP" {
		t.Fatalf("expected normalized TCP ports, ports=%+v err=%v", ports, err)
	}
}
