package model

import (
	"testing"
)

func TestNewProfile(t *testing.T) {
	p := NewProfile("user-001", "John Doe", "john@nexus.com")

	if p.UserID != "user-001" {
		t.Errorf("expected UserID 'user-001', got '%s'", p.UserID)
	}
	if p.DisplayName != "John Doe" {
		t.Errorf("expected DisplayName 'John Doe', got '%s'", p.DisplayName)
	}
	if p.Email != "john@nexus.com" {
		t.Errorf("expected Email 'john@nexus.com', got '%s'", p.Email)
	}
	if p.MembershipTier != MembershipTierBronze {
		t.Errorf("expected default tier BRONZE, got '%s'", p.MembershipTier)
	}
	if p.LoyaltyPoints != 0 {
		t.Errorf("expected 0 loyalty points, got %d", p.LoyaltyPoints)
	}
	if p.HasShop() {
		t.Error("expected new profile to have no shop")
	}

	// Add ShopConfig
	p.ShopConfig = &ShopConfig{
		ShopID:   "shop-101",
		ShopName: "Nexus Official",
	}
	if !p.HasShop() {
		t.Error("expected profile to have shop after setting ShopConfig")
	}
}

func TestMembershipTiers(t *testing.T) {
	tiers := []MembershipTier{
		MembershipTierBronze,
		MembershipTierSilver,
		MembershipTierGold,
		MembershipTierPlatinum,
		MembershipTierDiamond,
	}

	for _, tier := range tiers {
		if tier == "" {
			t.Error("expected non-empty membership tier")
		}
	}
}
