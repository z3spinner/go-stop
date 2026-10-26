// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestNotificationPreferenceRepo_DefaultsSetAndOptOuts(t *testing.T) {
	truncate(t)
	repo := postgres.NewNotificationPreferenceRepo(testPool)

	p, err := repo.Get("0698000001")
	if err != nil || !p.MorningDigest {
		t.Fatalf("no row: want the default (digest on), got %+v, %v", p, err)
	}
	if err := repo.SetMorningDigest("0698000001", false); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetMorningDigest("0698000002", false); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetMorningDigest("0698000002", true); err != nil { // upsert back on
		t.Fatal(err)
	}
	if p, _ := repo.Get("0698000001"); p.MorningDigest {
		t.Error("expected 0698000001 off")
	}
	out, err := repo.MorningDigestOptOuts()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "0698000001" {
		t.Errorf("opt-outs = %v, want [0698000001]", out)
	}
}
