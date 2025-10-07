package managers

import (
	"context"
	"encoding/json"
	"fmt"
)

// MinioPolicyInfo represents MinIO policy information
type MinioPolicyInfo struct {
	Name      string                 `json:"name"`
	Policy    map[string]interface{} `json:"policy"`
	CreatedAt string                 `json:"created_at,omitempty"`
}

// CreatePolicy creates a new MinIO policy
func (m *MinIOManager) CreatePolicy(ctx context.Context, policyName string, policyDocument map[string]interface{}) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Convert policy document to JSON bytes
	policyBytes, err := json.Marshal(policyDocument)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	// Create policy
	if err := adminClient.AddCannedPolicy(ctx, policyName, policyBytes); err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// DeletePolicy deletes a MinIO policy
func (m *MinIOManager) DeletePolicy(ctx context.Context, policyName string) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Remove policy
	if err := adminClient.RemoveCannedPolicy(ctx, policyName); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}

// ListPolicies lists all MinIO policies
func (m *MinIOManager) ListPolicies(ctx context.Context) ([]MinioPolicyInfo, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin client: %w", err)
	}

	// List policies
	policies, err := adminClient.ListCannedPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	var policyList []MinioPolicyInfo
	for name, policyBytes := range policies {
		var policyDoc map[string]interface{}
		if err := json.Unmarshal(policyBytes, &policyDoc); err != nil {
			// Skip invalid policy documents
			continue
		}

		policyInfo := MinioPolicyInfo{
			Name:   name,
			Policy: policyDoc,
		}
		policyList = append(policyList, policyInfo)
	}

	return policyList, nil
}

// GetPolicy retrieves a specific policy
func (m *MinIOManager) GetPolicy(ctx context.Context, policyName string) (*MinioPolicyInfo, error) {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin client: %w", err)
	}

	// Get policy
	policyBytes, err := adminClient.InfoCannedPolicy(ctx, policyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	var policyDoc map[string]interface{}
	if err := json.Unmarshal(policyBytes, &policyDoc); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	policyInfo := &MinioPolicyInfo{
		Name:   policyName,
		Policy: policyDoc,
	}

	return policyInfo, nil
}

// UpdatePolicy updates an existing policy
func (m *MinIOManager) UpdatePolicy(ctx context.Context, policyName string, policyDocument map[string]interface{}) error {
	adminClient, err := m.getAdminClient()
	if err != nil {
		return fmt.Errorf("failed to get admin client: %w", err)
	}

	// Convert policy document to JSON bytes
	policyBytes, err := json.Marshal(policyDocument)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	// Update policy (same as create, it overwrites)
	if err := adminClient.AddCannedPolicy(ctx, policyName, policyBytes); err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

// GetDefaultPolicies returns a list of built-in MinIO policies
func (m *MinIOManager) GetDefaultPolicies() []MinioPolicyInfo {
	return []MinioPolicyInfo{
		{
			Name: "readonly",
			Policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect": "Allow",
						"Action": []string{
							"s3:GetBucketLocation",
							"s3:GetObject",
						},
						"Resource": []string{
							"arn:aws:s3:::*",
						},
					},
				},
			},
		},
		{
			Name: "writeonly",
			Policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect": "Allow",
						"Action": []string{
							"s3:PutObject",
						},
						"Resource": []string{
							"arn:aws:s3:::*",
						},
					},
				},
			},
		},
		{
			Name: "readwrite",
			Policy: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect": "Allow",
						"Action": []string{
							"s3:*",
						},
						"Resource": []string{
							"arn:aws:s3:::*",
						},
					},
				},
			},
		},
	}
}

// ValidatePolicy validates a policy document structure
func (m *MinIOManager) ValidatePolicy(policyDocument map[string]interface{}) error {
	// Check for required fields
	version, ok := policyDocument["Version"].(string)
	if !ok {
		return fmt.Errorf("policy must have a Version field")
	}

	if version != "2012-10-17" {
		return fmt.Errorf("unsupported policy version: %s", version)
	}

	statements, ok := policyDocument["Statement"]
	if !ok {
		return fmt.Errorf("policy must have a Statement field")
	}

	// Validate statements array
	stmtArray, ok := statements.([]interface{})
	if !ok {
		return fmt.Errorf("Statement must be an array")
	}

	if len(stmtArray) == 0 {
		return fmt.Errorf("Statement array cannot be empty")
	}

	// Validate each statement
	for i, stmt := range stmtArray {
		stmtMap, ok := stmt.(map[string]interface{})
		if !ok {
			return fmt.Errorf("statement %d must be an object", i)
		}

		// Check required fields in statement
		if _, ok := stmtMap["Effect"]; !ok {
			return fmt.Errorf("statement %d must have Effect field", i)
		}

		if _, ok := stmtMap["Action"]; !ok {
			return fmt.Errorf("statement %d must have Action field", i)
		}

		if _, ok := stmtMap["Resource"]; !ok {
			return fmt.Errorf("statement %d must have Resource field", i)
		}
	}

	return nil
}

// GenerateBucketPolicy generates a policy for specific bucket access
func (m *MinIOManager) GenerateBucketPolicy(bucketName string, accessLevel string) (map[string]interface{}, error) {
	var actions []string

	switch accessLevel {
	case "readonly":
		actions = []string{
			"s3:GetBucketLocation",
			"s3:GetObject",
			"s3:ListBucket",
		}
	case "writeonly":
		actions = []string{
			"s3:PutObject",
		}
	case "readwrite":
		actions = []string{
			"s3:GetBucketLocation",
			"s3:GetObject",
			"s3:PutObject",
			"s3:DeleteObject",
			"s3:ListBucket",
		}
	default:
		return nil, fmt.Errorf("invalid access level: %s", accessLevel)
	}

	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": actions,
				"Resource": []string{
					fmt.Sprintf("arn:aws:s3:::%s", bucketName),
					fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
			},
		},
	}

	return policy, nil
}

// ListUserPolicies lists policies attached to a user
func (m *MinIOManager) ListUserPolicies(ctx context.Context, accessKey string) ([]string, error) {
	userInfo, err := m.GetUserInfo(ctx, accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	policies := []string{}
	if userInfo.Policy != "" {
		policies = append(policies, userInfo.Policy)
	}

	return policies, nil
}
