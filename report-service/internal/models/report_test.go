package models

import "testing"

func TestCategoryConstants(t *testing.T) {
	if CategoryModeration != "moderation" {
		t.Fatalf("CategoryModeration = %q", CategoryModeration)
	}
	if CategoryBug != "bug" {
		t.Fatalf("CategoryBug = %q", CategoryBug)
	}
}

func TestEntityConstants(t *testing.T) {
	if EntityPost != "post" {
		t.Fatalf("EntityPost = %q", EntityPost)
	}
	if EntityProfile != "profile" {
		t.Fatalf("EntityProfile = %q", EntityProfile)
	}
	if EntityApp != "app" {
		t.Fatalf("EntityApp = %q", EntityApp)
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusOpen != "open" {
		t.Fatalf("StatusOpen = %q", StatusOpen)
	}
	if StatusClosed != "closed" {
		t.Fatalf("StatusClosed = %q", StatusClosed)
	}
	if StatusReopened != "reopened" {
		t.Fatalf("StatusReopened = %q", StatusReopened)
	}
	if StatusApproved != "approved" {
		t.Fatalf("StatusApproved = %q", StatusApproved)
	}
}

func TestReasonConstants(t *testing.T) {
	if ReasonInappropriate != "inappropriate" {
		t.Fatalf("ReasonInappropriate = %q", ReasonInappropriate)
	}
	if ReasonOffensive != "offensive" {
		t.Fatalf("ReasonOffensive = %q", ReasonOffensive)
	}
	if ReasonBug != "bug" {
		t.Fatalf("ReasonBug = %q", ReasonBug)
	}
	if ReasonSpam != "spam" {
		t.Fatalf("ReasonSpam = %q", ReasonSpam)
	}
	if ReasonOther != "other" {
		t.Fatalf("ReasonOther = %q", ReasonOther)
	}
}

func TestSettingsSingletonID(t *testing.T) {
	if SettingsSingletonID != "global" {
		t.Fatalf("SettingsSingletonID = %q", SettingsSingletonID)
	}
}

func TestDefaultAutoHideThreshold(t *testing.T) {
	if DefaultAutoHideThreshold != 5 {
		t.Fatalf("DefaultAutoHideThreshold = %d, attendu 5", DefaultAutoHideThreshold)
	}
}
