package connection_history

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "lazypg"
)

// PasswordStore handles secure password storage using OS keyring
type PasswordStore struct {
	service string
}

// NewPasswordStore creates a new password store
func NewPasswordStore() *PasswordStore {
	return &PasswordStore{
		service: serviceName,
	}
}

// Save stores a password securely in the OS keyring
// key format: "host:port:database:user" for uniqueness
func (ps *PasswordStore) Save(host string, port int, database, user, password string) error {
	if password == "" {
		// Don't save empty passwords
		return nil
	}

	key := makeKey(host, port, database, user)
	return keyring.Set(ps.service, key, password)
}

// Get retrieves a password from the OS keyring
func (ps *PasswordStore) Get(host string, port int, database, user string) (string, error) {
	key := makeKey(host, port, database, user)
	password, err := keyring.Get(ps.service, key)
	if err != nil {
		// Password not found is not an error, just return empty
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get password from keyring: %w", err)
	}
	return password, nil
}

// Delete removes a password from the OS keyring
func (ps *PasswordStore) Delete(host string, port int, database, user string) error {
	key := makeKey(host, port, database, user)
	err := keyring.Delete(ps.service, key)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete password from keyring: %w", err)
	}
	return nil
}

// makeKey creates a unique key for password storage
func makeKey(host string, port int, database, user string) string {
	return fmt.Sprintf("%s:%d:%s:%s", host, port, database, user)
}
