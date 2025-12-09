package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPodmanRunnerStub(t *testing.T) {
	// Test that the stub implementation works
	repoInfo := repoInfoWithProjectRoot(t)
	repoInfo.Slug = "BADWOLF"
	runner := newPodmanShellRunner(true, nil, repoInfo)
	require.NotNil(t, runner)
	require.Equal(t, "localhost/asimi-sandbox-BADWOLF:latest", runner.imageName)
	require.True(t, runner.allowFallback)
}
