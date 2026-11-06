package store

import (
	"context"
	"testing"
)

func TestSeedOnlyAppliesBeforeTheFirstSave(t *testing.T) {
	db, _ := seeded(t)
	ctx := context.Background()
	if err := db.SeedNtfy(ctx, NtfySettings{Server: "https://a.test", Topic: "env"}); err != nil {
		t.Fatal(err)
	}
	// Clearing the topic on screen must survive a restart with the old environment.
	if err := db.SaveNtfy(ctx, NtfySettings{Server: "https://a.test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.SeedNtfy(ctx, NtfySettings{Server: "https://a.test", Topic: "env"}); err != nil {
		t.Fatal(err)
	}
	settings, err := db.Ntfy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Topic != "" {
		t.Fatalf("settings: %+v", settings)
	}
}
