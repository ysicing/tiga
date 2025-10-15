package minio

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidatePolicy_Success(t *testing.T) {
	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{map[string]interface{}{
			"Effect": "Allow", "Action": []string{"s3:GetObject"}, "Resource": []string{"arn:aws:s3:::b/*"},
		}},
	}
	if err := ValidatePolicy(doc); err != nil {
		t.Fatalf("ValidatePolicy() unexpected error: %v", err)
	}
}

func TestValidatePolicy_Fail(t *testing.T) {
	doc := map[string]interface{}{"Statement": []interface{}{}}
	if err := ValidatePolicy(doc); err == nil {
		t.Fatalf("expected error for missing Version")
	}
}

func TestGenerateBucketPolicy(t *testing.T) {
	doc, err := GenerateBucketPolicy("mybucket", "readonly")
	if err != nil {
		t.Fatalf("GenerateBucketPolicy error: %v", err)
	}
	by, _ := json.Marshal(doc)
	s := string(by)
	if !(strings.Contains(s, "arn:aws:s3:::mybucket") && strings.Contains(s, "s3:GetObject")) {
		t.Fatalf("unexpected policy content: %s", s)
	}
}

func TestGeneratePrefixPolicy(t *testing.T) {
	doc, err := GeneratePrefixPolicy("bkt", "prefix/", "readwrite")
	if err != nil {
		t.Fatalf("GeneratePrefixPolicy error: %v", err)
	}
	by, _ := json.Marshal(doc)
	s := string(by)
	if !(strings.Contains(s, "arn:aws:s3:::bkt/prefix/*") && strings.Contains(s, "s3:PutObject") && strings.Contains(s, "s3:GetObject")) {
		t.Fatalf("unexpected policy content: %s", s)
	}
}

// helpers not needed; use strings.Contains
