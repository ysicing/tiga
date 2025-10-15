package minio

import (
	"encoding/json"
	"fmt"
)

func ValidatePolicy(doc map[string]interface{}) error {
	if v, ok := doc["Version"].(string); !ok || v == "" {
		return fmt.Errorf("policy must have Version")
	}
	if _, ok := doc["Statement"]; !ok {
		return fmt.Errorf("policy must have Statement")
	}
	if arr, ok := doc["Statement"].([]interface{}); !ok || len(arr) == 0 {
		return fmt.Errorf("Statement must be non-empty array")
	}
	return nil
}

func GenerateBucketPolicy(bucket, permission string) (map[string]interface{}, error) {
	actions, err := actionsFor(permission)
	if err != nil {
		return nil, err
	}
	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{"Effect": "Allow", "Action": actions, "Resource": []string{fmt.Sprintf("arn:aws:s3:::%s", bucket), fmt.Sprintf("arn:aws:s3:::%s/*", bucket)}},
		},
	}
	return doc, nil
}

func GeneratePrefixPolicy(bucket, prefix, permission string) (map[string]interface{}, error) {
	actions, err := actionsFor(permission)
	if err != nil {
		return nil, err
	}
	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{"Effect": "Allow", "Action": actions, "Resource": []string{fmt.Sprintf("arn:aws:s3:::%s", bucket), fmt.Sprintf("arn:aws:s3:::%s/%s*", bucket, prefix)}},
		},
	}
	return doc, nil
}

func actionsFor(permission string) ([]string, error) {
	switch permission {
	case "readonly":
		return []string{"s3:GetBucketLocation", "s3:GetObject", "s3:ListBucket"}, nil
	case "writeonly":
		return []string{"s3:PutObject"}, nil
	case "readwrite":
		return []string{"s3:GetBucketLocation", "s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucket"}, nil
	default:
		return nil, fmt.Errorf("invalid permission: %s", permission)
	}
}

func MarshalPolicy(doc map[string]interface{}) (string, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
