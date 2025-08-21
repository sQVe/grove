package config

import (
	"os"
	"testing"
)

func TestGlobalConfig(t *testing.T) {
	t.Run("initial state is false for both modes", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		if IsPlain() {
			t.Error("Expected plain mode to be false initially")
		}
		if IsDebug() {
			t.Error("Expected debug mode to be false initially")
		}
	})

	t.Run("plain mode can be enabled", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		Global.Plain = true
		if !IsPlain() {
			t.Error("Expected plain mode to be true after setting Global.Plain = true")
		}
	})

	t.Run("debug mode can be enabled", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		Global.Debug = true
		if !IsDebug() {
			t.Error("Expected debug mode to be true after setting Global.Debug = true")
		}
	})
}

func TestLoadFromEnv(t *testing.T) {
	t.Run("both false with no env vars", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		LoadFromEnv()
		if IsPlain() || IsDebug() {
			t.Error("Expected both modes to be false with no env vars")
		}
	})

	t.Run("plain true with GROVE_PLAIN=1", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_DEBUG")
		_ = os.Setenv("GROVE_PLAIN", "1")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromEnv()
		if !IsPlain() {
			t.Error("Expected plain mode to be true with GROVE_PLAIN=1")
		}
	})

	t.Run("debug true with GROVE_DEBUG=1", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Setenv("GROVE_DEBUG", "1")
		defer func() { _ = os.Unsetenv("GROVE_DEBUG") }()

		LoadFromEnv()
		if !IsDebug() {
			t.Error("Expected debug mode to be true with GROVE_DEBUG=1")
		}
	})

	t.Run("plain true with GROVE_PLAIN=true", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_DEBUG")
		_ = os.Setenv("GROVE_PLAIN", "true")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromEnv()
		if !IsPlain() {
			t.Error("Expected plain mode to be true with GROVE_PLAIN=true")
		}
	})

	t.Run("debug true with GROVE_DEBUG=true", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Setenv("GROVE_DEBUG", "true")
		defer func() { _ = os.Unsetenv("GROVE_DEBUG") }()

		LoadFromEnv()
		if !IsDebug() {
			t.Error("Expected debug mode to be true with GROVE_DEBUG=true")
		}
	})

	t.Run("both false with invalid env values", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Setenv("GROVE_PLAIN", "yes")
		_ = os.Setenv("GROVE_DEBUG", "on")
		defer func() {
			_ = os.Unsetenv("GROVE_PLAIN")
			_ = os.Unsetenv("GROVE_DEBUG")
		}()

		LoadFromEnv()
		if IsPlain() || IsDebug() {
			t.Error("Expected both modes to be false with invalid env values")
		}
	})
}
