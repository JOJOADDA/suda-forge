package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrBootstrapComplete  = errors.New("administrator already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const (
	passwordSaltBytes = 16
	passwordKeyBytes  = 32
	argonTime         = 3
	argonMemory       = 64 * 1024
	argonThreads      = 2
)

type UserRepository interface {
	CountUsers(ctx context.Context) (int, error)
	CreateAdmin(ctx context.Context, user User, passwordHash string) error
	FindByEmail(ctx context.Context, email string) (User, string, error)
	FindByID(ctx context.Context, id string) (User, error)
	TouchLogin(ctx context.Context, id string, now time.Time) error
}

type PostgresUserRepository struct {
	DB *pgxpool.Pool
}

func (r PostgresUserRepository) CountUsers(ctx context.Context) (int, error) {
	if r.DB == nil {
		return 0, errors.New("database pool is required")
	}
	var count int
	err := r.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r PostgresUserRepository) CreateAdmin(ctx context.Context, user User, passwordHash string) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('suda-forge-bootstrap'))`); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrBootstrapComplete
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, email, display_name, global_role, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'admin', 'active', $4, $4)
	`, user.ID, normalizeEmail(user.Email), user.DisplayName, user.CreatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_credentials (user_id, password_hash, password_changed_at)
		VALUES ($1, $2, $3)
	`, user.ID, passwordHash, user.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r PostgresUserRepository) FindByEmail(ctx context.Context, email string) (User, string, error) {
	if r.DB == nil {
		return User{}, "", errors.New("database pool is required")
	}
	var user User
	var passwordHash string
	err := r.DB.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.global_role, u.status,
		       u.created_at, u.updated_at, u.last_login_at, COALESCE(pc.password_hash, '')
		FROM users u
		LEFT JOIN password_credentials pc ON pc.user_id = u.id
		WHERE lower(u.email) = lower($1)
	`, normalizeEmail(email)).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &passwordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrUserNotFound
	}
	if err != nil {
		return User{}, "", err
	}
	return user, passwordHash, nil
}

func (r PostgresUserRepository) FindByID(ctx context.Context, id string) (User, error) {
	if r.DB == nil {
		return User{}, errors.New("database pool is required")
	}
	var user User
	err := r.DB.QueryRow(ctx, `
		SELECT id, email, display_name, global_role, status, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

func (r PostgresUserRepository) TouchLogin(ctx context.Context, id string, now time.Time) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `UPDATE users SET last_login_at=$2, updated_at=$2 WHERE id=$1`, id, now)
	return err
}

type AuthService struct {
	Users       UserRepository
	Sessions    SessionService
	Memberships MembershipRepository
	Now         func() time.Time
}

func (s AuthService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s AuthService) Bootstrap(ctx context.Context, email, displayName, password string) (User, error) {
	if s.Users == nil {
		return User{}, errors.New("user repository is required")
	}
	if err := validateEmail(email); err != nil {
		return User{}, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}
	count, err := s.Users.CountUsers(ctx)
	if err != nil {
		return User{}, err
	}
	if count > 0 {
		return User{}, ErrBootstrapComplete
	}
	now := s.now()
	user := User{ID: newUserID(), Email: normalizeEmail(email), DisplayName: strings.TrimSpace(displayName), Role: RoleAdmin, Status: "active", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	if err := s.Users.CreateAdmin(ctx, user, passwordHash); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (User, Session, string, error) {
	if s.Users == nil {
		return User{}, Session{}, "", errors.New("user repository is required")
	}
	user, passwordHash, err := s.Users.FindByEmail(ctx, email)
	if err != nil || user.Status != "active" || passwordHash == "" || !VerifyPassword(passwordHash, password) {
		return User{}, Session{}, "", ErrInvalidCredentials
	}
	if err := s.Users.TouchLogin(ctx, user.ID, s.now()); err != nil {
		return User{}, Session{}, "", err
	}
	session, token, err := s.Sessions.Create(ctx, SessionInput{UserID: user.ID, UserAgent: userAgent, IPAddress: ip})
	if err != nil {
		return User{}, Session{}, "", err
	}
	return user, session, token, nil
}

func (s AuthService) CurrentUser(ctx context.Context, session Session) (User, error) {
	if s.Users == nil {
		return User{}, errors.New("user repository is required")
	}
	return s.Users.FindByID(ctx, session.UserID)
}

func HashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, passwordKeyBytes)
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%x$%x", argonMemory, argonTime, argonThreads, salt, key), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return false
	}
	var memory, iterations, threads uint32
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil || memory == 0 || iterations == 0 || threads == 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return false
	}
	expected, err := hex.DecodeString(parts[4])
	if err != nil || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, uint8(threads), uint32(len(expected)))
	if len(actual) != len(expected) {
		return false
	}
	var diff byte
	for i := range actual {
		diff |= actual[i] ^ expected[i]
	}
	return diff == 0
}

func validateEmail(email string) error {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") || strings.ContainsAny(email, "\r\n") {
		return errors.New("a valid email is required")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func newUserID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("user-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("user-%x", buf)
}
