package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInitDBOptionsFromArgsReadsSeedFlag(t *testing.T) {
	t.Setenv("INITDB_FRESH", "")
	t.Setenv("INITDB_SEED", "")
	t.Setenv("INITDB_ALLOW_PRODUCTION_FRESH", "")

	opts := parseInitDBOptionsFromArgs([]string{"--fresh", "--seed", "--allow-production-fresh"})

	require.True(t, opts.fresh)
	require.True(t, opts.seed)
	require.True(t, opts.allowProductionFresh)
}

func TestParseInitDBOptionsFromArgsReadsEnvFallback(t *testing.T) {
	t.Setenv("INITDB_FRESH", "true")
	t.Setenv("INITDB_SEED", "1")
	t.Setenv("INITDB_ALLOW_PRODUCTION_FRESH", "yes")

	opts := parseInitDBOptionsFromArgs(nil)

	require.True(t, opts.fresh)
	require.True(t, opts.seed)
	require.True(t, opts.allowProductionFresh)
}

func TestEnvIntHelpersFallbackOnInvalidValue(t *testing.T) {
	key := "DUMMY_SEED_ADMIN_COUNT"
	t.Setenv(key, "nope")

	require.Equal(t, 50, envIntOrDefault(key, 50))

	t.Setenv(key, "73")
	require.Equal(t, 73, envIntOrDefault(key, 50))
}

func TestEnvInt64HelpersFallbackOnInvalidValue(t *testing.T) {
	key := "DUMMY_SEED_RANDOM_SEED"
	t.Setenv(key, "invalid")

	require.EqualValues(t, 20260321, envInt64OrDefault(key, 20260321))

	t.Setenv(key, "99")
	require.EqualValues(t, 99, envInt64OrDefault(key, 20260321))
}
