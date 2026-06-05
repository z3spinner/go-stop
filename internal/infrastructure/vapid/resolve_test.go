// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package vapid

import (
	"context"
	"testing"
)

// fakeStore is an in-memory Store for unit testing Resolve without a DB.
type fakeStore struct{ m map[string]string }

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]string{}} }

func (s *fakeStore) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := s.m[key]
	return v, ok, nil
}

func (s *fakeStore) InsertIfAbsent(_ context.Context, key, value string) error {
	if _, ok := s.m[key]; !ok {
		s.m[key] = value
	}
	return nil
}

func emptyEnv(string) string { return "" }

func TestResolve_GeneratesAndPersistsWhenEmpty(t *testing.T) {
	// Deterministic key generation for the test.
	orig := generateKeys
	generateKeys = func() (string, string, error) { return "PRIV", "PUB", nil }
	defer func() { generateKeys = orig }()

	store := newFakeStore()
	keys, source, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "generated" {
		t.Fatalf("source = %q, want \"generated\"", source)
	}
	if keys.Public != "PUB" || keys.Private != "PRIV" {
		t.Fatalf("keys = %+v, want PUB/PRIV", keys)
	}
	if keys.Email != defaultEmail {
		t.Fatalf("email = %q, want %q", keys.Email, defaultEmail)
	}

	// Second call returns the SAME keys from the store, source "db".
	keys2, source2, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve (2nd): %v", err)
	}
	if source2 != "db" {
		t.Fatalf("source2 = %q, want \"db\"", source2)
	}
	if keys2 != keys {
		t.Fatalf("keys changed across calls: %+v vs %+v", keys2, keys)
	}
}

func TestResolve_AdoptsEnvThenDBOnly(t *testing.T) {
	store := newFakeStore()
	env := func(k string) string {
		switch k {
		case "VAPID_PUBLIC_KEY":
			return "ENVPUB"
		case "VAPID_PRIVATE_KEY":
			return "ENVPRIV"
		case "VAPID_EMAIL":
			return "mailto:env@example.com"
		}
		return ""
	}

	keys, source, err := Resolve(context.Background(), store, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "env" {
		t.Fatalf("source = %q, want \"env\"", source)
	}
	if keys.Public != "ENVPUB" || keys.Private != "ENVPRIV" || keys.Email != "mailto:env@example.com" {
		t.Fatalf("keys = %+v, want env values", keys)
	}

	// With env now cleared, the same keys still come from the DB.
	keys2, source2, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve (2nd): %v", err)
	}
	if source2 != "db" {
		t.Fatalf("source2 = %q, want \"db\"", source2)
	}
	if keys2 != keys {
		t.Fatalf("keys changed: %+v vs %+v", keys2, keys)
	}
}

func TestResolve_DBWinsOverEnv(t *testing.T) {
	store := newFakeStore()
	store.m["vapid_public"] = "DBPUB"
	store.m["vapid_private"] = "DBPRIV"
	store.m["vapid_email"] = "mailto:db@example.com"

	env := func(k string) string {
		if k == "VAPID_PUBLIC_KEY" {
			return "ENVPUB"
		}
		if k == "VAPID_PRIVATE_KEY" {
			return "ENVPRIV"
		}
		return ""
	}

	keys, source, err := Resolve(context.Background(), store, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "db" {
		t.Fatalf("source = %q, want \"db\"", source)
	}
	if keys.Public != "DBPUB" || keys.Private != "DBPRIV" {
		t.Fatalf("keys = %+v, want DB values", keys)
	}
}
