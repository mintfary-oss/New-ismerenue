package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/mintfary/aqi-platform/internal/domain"
)

// newTestUserService — вспомогательная функция.
func newTestUserService(repo *mockUserRepo) *UserService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	auth := newTestAuthService(repo)
	return NewUserService(repo, auth, logger)
}

// TestUserCreate проверяет создание нового пользователя.
func TestUserCreate(t *testing.T) {
	ctx := context.Background()
	svc := newTestUserService(newMockUserRepo())

	u, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "new@example.com",
		Username: "newuser",
		Password: "SecurePassword1!",
		Role:     domain.RoleAnalyst,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if u.Email != "new@example.com" {
		t.Errorf("Create() email = %q, хотим %q", u.Email, "new@example.com")
	}
	if u.Password == "SecurePassword1!" {
		t.Error("Create() пароль не должен храниться в открытом виде")
	}
	if !u.IsActive {
		t.Error("Create() новый пользователь должен быть активен")
	}
	if u.ID.String() == "" {
		t.Error("Create() ID не должен быть пустым")
	}
}

// TestUserCreateDuplicateEmail проверяет конфликт при повторном email.
func TestUserCreateDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	svc := newTestUserService(newMockUserRepo())

	input := domain.CreateUserInput{
		Email:    "dup@example.com",
		Username: "dupuser",
		Password: "Pass123!",
		Role:     domain.RoleAnalyst,
	}

	_, err := svc.Create(ctx, input)
	mustNotError(err)

	// Второй с тем же email должен вернуть конфликт
	_, err = svc.Create(ctx, input)
	if err == nil {
		t.Fatal("Create() с дублирующимся email должен вернуть ошибку")
	}
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("Create() ошибка = %v, хотим ErrAlreadyExists", err)
	}
}

// TestUserCreateInvalidRole проверяет отклонение недопустимой роли.
func TestUserCreateInvalidRole(t *testing.T) {
	ctx := context.Background()
	svc := newTestUserService(newMockUserRepo())

	_, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "bad@example.com",
		Username: "baduser",
		Password: "Pass123!",
		Role:     domain.Role("superadmin"), // несуществующая роль
	})
	if err == nil {
		t.Fatal("Create() с недопустимой ролью должен вернуть ошибку")
	}
}

// TestUserList проверяет список пользователей.
func TestUserList(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := newTestUserService(repo)

	// Создаём 3 пользователей
	for i := range 3 {
		suffix := string(rune('a' + i))
		_, err := svc.Create(ctx, domain.CreateUserInput{
			Email:    "user" + suffix + "@example.com",
			Username: "user" + suffix,
			Password: "Pass123!",
			Role:     domain.RoleAnalyst,
		})
		mustNotError(err)
	}

	users, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("List() total = %d, хотим 3", total)
	}
	if len(users) != 3 {
		t.Errorf("List() len = %d, хотим 3", len(users))
	}
}

// TestUserDelete проверяет деактивацию пользователя (soft delete).
func TestUserDelete(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := newTestUserService(repo)

	u, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "del@example.com",
		Username: "deluser",
		Password: "Pass123!",
		Role:     domain.RoleAnalyst,
	})
	mustNotError(err)

	// Удаляем
	if err := svc.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Пользователь должен быть деактивирован (soft delete)
	updated, err := svc.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID() после Delete() error = %v", err)
	}
	if updated.IsActive {
		t.Error("Delete() пользователь должен быть деактивирован (is_active=false)")
	}
}

// TestUserUpdate проверяет обновление данных пользователя.
func TestUserUpdate(t *testing.T) {
	ctx := context.Background()
	svc := newTestUserService(newMockUserRepo())

	u, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "upd@example.com",
		Username: "oldname",
		Password: "Pass123!",
		Role:     domain.RoleAnalyst,
	})
	mustNotError(err)

	newUsername := "newname"
	newRole := domain.RoleAdmin
	updated, err := svc.Update(ctx, u.ID, domain.UpdateUserInput{
		Username: &newUsername,
		Role:     &newRole,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Username != "newname" {
		t.Errorf("Update() Username = %q, хотим %q", updated.Username, "newname")
	}
	if updated.Role != domain.RoleAdmin {
		t.Errorf("Update() Role = %q, хотим %q", updated.Role, domain.RoleAdmin)
	}
}

// TestChangePasswordSuccess проверяет успешную смену пароля.
func TestChangePasswordSuccess(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := newTestUserService(repo)

	u, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "chpw@example.com",
		Username: "chpwuser",
		Password: "OldPassword!",
		Role:     domain.RoleAnalyst,
	})
	mustNotError(err)

	err = svc.ChangePassword(ctx, u.ID, domain.ChangePasswordInput{
		OldPassword: "OldPassword!",
		NewPassword: "NewPassword!",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	// Новый пароль должен работать при логине
	authSvc := newTestAuthService(repo)
	_, err = authSvc.Login(ctx, LoginInput{
		Email:    "chpw@example.com",
		Password: "NewPassword!",
	})
	if err != nil {
		t.Fatalf("Login() после смены пароля error = %v", err)
	}
}

// TestChangePasswordWrongOld проверяет отказ при неверном старом пароле.
func TestChangePasswordWrongOld(t *testing.T) {
	ctx := context.Background()
	svc := newTestUserService(newMockUserRepo())

	u, err := svc.Create(ctx, domain.CreateUserInput{
		Email:    "chpw2@example.com",
		Username: "chpw2user",
		Password: "RealOldPass!",
		Role:     domain.RoleAnalyst,
	})
	mustNotError(err)

	err = svc.ChangePassword(ctx, u.ID, domain.ChangePasswordInput{
		OldPassword: "WrongOldPass!",
		NewPassword: "NewPassword!",
	})
	if err == nil {
		t.Fatal("ChangePassword() с неверным старым паролем должен вернуть ошибку")
	}
}
