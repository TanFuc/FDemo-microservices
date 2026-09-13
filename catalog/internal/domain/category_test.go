package domain

import (
	"testing"
)

func TestCategory_EnsureDefaults(t *testing.T) {
	cat := &Category{
		Slug: "electronics",
	}

	cat.EnsureDefaults()

	if cat.Status != CategoryStatusActive {
		t.Errorf("expected status '%s', got '%s'", CategoryStatusActive, cat.Status)
	}
	if !cat.IsVisible {
		t.Error("expected IsVisible to be true")
	}
	if cat.Name == nil {
		t.Error("expected Name map to be initialized")
	}
	if cat.Description == nil {
		t.Error("expected Description map to be initialized")
	}
	if cat.AttributeDefinitions == nil {
		t.Error("expected AttributeDefinitions slice to be initialized")
	}
}

func TestCategory_StatusConstants(t *testing.T) {
	if CategoryStatusActive != "ACTIVE" {
		t.Errorf("expected 'ACTIVE', got '%s'", CategoryStatusActive)
	}
	if CategoryStatusInactive != "INACTIVE" {
		t.Errorf("expected 'INACTIVE', got '%s'", CategoryStatusInactive)
	}
	if CategoryStatusHidden != "HIDDEN" {
		t.Errorf("expected 'HIDDEN', got '%s'", CategoryStatusHidden)
	}
}
