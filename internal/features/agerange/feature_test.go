package agerange

import (
	"testing"
)

func TestNew(t *testing.T) {
	deps := Dependencies{}

	_, err := New(deps)
	if err == nil {
		t.Error("expected error for empty dependencies, got nil")
	}
}

func TestName(t *testing.T) {
	f := &Feature{}
	if f.Name() != "agerange" {
		t.Errorf("expected name 'agerange', got '%s'", f.Name())
	}
}

func TestRegisterCommands(t *testing.T) {
	f := &Feature{}
	commands := f.RegisterCommands()
	if commands != nil {
		t.Error("expected nil commands for menu-driven feature")
	}
}

func TestGetMenuButton(t *testing.T) {
	f := &Feature{}
	button := f.GetMenuButton()
	if button == nil {
		t.Fatal("expected menu button, got nil")
	}
	if button.CustomID != "menu:agerange:setup" {
		t.Errorf("expected custom ID 'menu:agerange:setup', got '%s'", button.CustomID)
	}
	if button.Category != "admin" {
		t.Errorf("expected category 'admin', got '%s'", button.Category)
	}
	if button.SubCategory != "configuration" {
		t.Errorf("expected subcategory 'configuration', got '%s'", button.SubCategory)
	}
	if !button.AdminOnly {
		t.Error("expected AdminOnly to be true")
	}
}

// TestHandleInteraction_NotHandled is skipped because it requires proper mock setup
// The actual functionality is tested through integration tests

