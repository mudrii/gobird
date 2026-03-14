package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPaths_HomeError(t *testing.T) {
	prevHome := userHomeDirFunc
	prevGetwd := getwdFunc
	userHomeDirFunc = func() (string, error) { return "", errors.New("boom") }
	getwdFunc = func() (string, error) { return "/tmp/work", nil }
	t.Cleanup(func() {
		userHomeDirFunc = prevHome
		getwdFunc = prevGetwd
	})

	_, err := defaultConfigPaths()
	if err == nil {
		t.Fatal("expected error when home directory lookup fails")
	}
}

func TestDefaultConfigPaths_GetwdErrorFallsBackToGlobal(t *testing.T) {
	prevHome := userHomeDirFunc
	prevGetwd := getwdFunc
	userHomeDirFunc = func() (string, error) { return "/tmp/home", nil }
	getwdFunc = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() {
		userHomeDirFunc = prevHome
		getwdFunc = prevGetwd
	})

	paths, err := defaultConfigPaths()
	if err != nil {
		t.Fatalf("defaultConfigPaths: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("want 1 fallback path, got %d", len(paths))
	}
	want := filepath.Join("/tmp/home", ".config", "gobird", "config.json5")
	if paths[0] != want {
		t.Fatalf("want %q, got %q", want, paths[0])
	}
}

func TestDefaultConfigPaths_UsesGobirdNames(t *testing.T) {
	prevHome := userHomeDirFunc
	prevGetwd := getwdFunc
	userHomeDirFunc = func() (string, error) { return "/tmp/home", nil }
	getwdFunc = func() (string, error) { return "/tmp/work", nil }
	t.Cleanup(func() {
		userHomeDirFunc = prevHome
		getwdFunc = prevGetwd
	})

	paths, err := defaultConfigPaths()
	if err != nil {
		t.Fatalf("defaultConfigPaths: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("want 2 paths, got %d", len(paths))
	}
	if paths[0] != filepath.Join("/tmp/home", ".config", "gobird", "config.json5") {
		t.Fatalf("unexpected global path: %q", paths[0])
	}
	if paths[1] != filepath.Join("/tmp/work", ".gobirdrc.json5") {
		t.Fatalf("unexpected local path: %q", paths[1])
	}
}
