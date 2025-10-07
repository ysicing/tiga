package managers

import (
	"context"
	"fmt"

	"github.com/minio/madmin-go/v3"
)

// MinIOUserInfo represents MinIO user information
type MinIOUserInfo struct {
	AccessKey string                 `json:"access_key"`
	Status    string                 `json:"status"`
	Policy    string                 `json:"policy"`
	MemberOf  []string               `json:"member_of,omitempty"`
	UpdatedAt string                 `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CreateUser creates a new MinIO user
func (m *MinIOManager) CreateUser(ctx context.Context, accessKey, secretKey string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Create user
	if err := adminClient.AddUser(ctx, accessKey, secretKey); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// DeleteUser deletes a MinIO user
func (m *MinIOManager) DeleteUser(ctx context.Context, accessKey string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Remove user
	if err := adminClient.RemoveUser(ctx, accessKey); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers lists all MinIO users
func (m *MinIOManager) ListUsers(ctx context.Context) ([]MinIOUserInfo, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin client: %w", err)
	}

	// List users
	users, err := adminClient.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var userList []MinIOUserInfo
	for accessKey, userInfo := range users {
		user := MinIOUserInfo{
			AccessKey: accessKey,
			Status:    string(userInfo.Status),
			Policy:    userInfo.PolicyName,
			MemberOf:  userInfo.MemberOf,
			UpdatedAt: userInfo.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
		userList = append(userList, user)
	}

	return userList, nil
}

// GetUserInfo retrieves information about a specific user
func (m *MinIOManager) GetUserInfo(ctx context.Context, accessKey string) (*MinIOUserInfo, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin client: %w", err)
	}

	// Get user info
	userInfo, err := adminClient.GetUserInfo(ctx, accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user := &MinIOUserInfo{
		AccessKey: accessKey,
		Status:    string(userInfo.Status),
		Policy:    userInfo.PolicyName,
		MemberOf:  userInfo.MemberOf,
		UpdatedAt: userInfo.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return user, nil
}

// SetUserStatus enables or disables a user
func (m *MinIOManager) SetUserStatus(ctx context.Context, accessKey string, enable bool) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	var status madmin.AccountStatus
	if enable {
		status = madmin.AccountEnabled
	} else {
		status = madmin.AccountDisabled
	}

	// Set user status
	if err := adminClient.SetUserStatus(ctx, accessKey, status); err != nil {
		return fmt.Errorf("failed to set user status: %w", err)
	}

	return nil
}

// AttachUserPolicy attaches a policy to a user
func (m *MinIOManager) AttachUserPolicy(ctx context.Context, accessKey, policyName string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Attach policy to user
	_, err = adminClient.AttachPolicy(ctx, madmin.PolicyAssociationReq{
		Policies: []string{policyName},
		User:     accessKey,
	})
	if err != nil {
		return fmt.Errorf("failed to attach policy: %w", err)
	}

	return nil
}

// DetachUserPolicy detaches a policy from a user
func (m *MinIOManager) DetachUserPolicy(ctx context.Context, accessKey, policyName string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Detach policy from user
	_, err = adminClient.DetachPolicy(ctx, madmin.PolicyAssociationReq{
		Policies: []string{policyName},
		User:     accessKey,
	})
	if err != nil {
		return fmt.Errorf("failed to detach policy: %w", err)
	}

	return nil
}

// CreateServiceAccount creates a service account for a user
func (m *MinIOManager) CreateServiceAccount(ctx context.Context, parentUser string, policy string) (string, string, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return "", "", fmt.Errorf("failed to get admin client: %w", err)
	}

	opts := madmin.AddServiceAccountReq{
		Policy: []byte(policy),
	}

	// Create service account
	creds, err := adminClient.AddServiceAccount(ctx, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to create service account: %w", err)
	}

	return creds.AccessKey, creds.SecretKey, nil
}

// DeleteServiceAccount deletes a service account
func (m *MinIOManager) DeleteServiceAccount(ctx context.Context, accessKey string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Delete service account
	if err := adminClient.DeleteServiceAccount(ctx, accessKey); err != nil {
		return fmt.Errorf("failed to delete service account: %w", err)
	}

	return nil
}

// ListServiceAccounts lists service accounts for a user
func (m *MinIOManager) ListServiceAccounts(ctx context.Context, user string) ([]string, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin client: %w", err)
	}

	// List service accounts
	accounts, err := adminClient.ListServiceAccounts(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	// Extract access keys
	accessKeys := make([]string, 0, len(accounts.Accounts))
	for _, account := range accounts.Accounts {
		accessKeys = append(accessKeys, account.AccessKey)
	}

	return accessKeys, nil
}
