package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "github.com/substratusai/sandboxai/go/api/v1"
	clientv1 "github.com/substratusai/sandboxai/go/client/v1"
)

func TestClientV1(t *testing.T) {
	httpc := &http.Client{Timeout: 60 * time.Second}
	c := clientv1.NewClient(cfg.SandboxAIBaseURL, clientv1.WithHTTPClient(httpc))

	// Sandbox lifecycle //

	sandboxJSON, err := os.ReadFile(os.Getenv("TEST_SANDBOX_PATH"))
	require.NoError(t, err, "reading sandbox json file")
	var spec v1.SandboxSpec
	require.NoError(t, json.Unmarshal(sandboxJSON, &spec))
	require.Equal(t, "placeholder-image", spec.Image)
	spec.Image = cfg.BoxImage

	const space = "default"
	createdSbx, err := c.CreateSandbox(ctx, space, &v1.CreateSandboxRequest{
		Spec: spec,
	})
	require.NoError(t, err, "Creating sandbox")
	require.NotEmpty(t, createdSbx.Name, "Sandbox name returned from create should not be empty")

	t.Cleanup(func() {
		t.Logf("Cleanup(): Deleting sandbox (space = %q, name = %q)", space, createdSbx.Name)
		err := c.DeleteSandbox(context.Background(), space, createdSbx.Name)
		require.NoError(t, err, "Failed to delete sandbox")
	})

	gottenSbx, err := c.GetSandbox(ctx, space, createdSbx.Name)
	require.NoError(t, err, "Getting sandbox")
	require.EqualValues(t, createdSbx, gottenSbx, "Sandbox returned from GetSandbox should match that returned from CreateSandbox")

	// IPython Tool //

	var ipyCases []struct {
		Name                   string `json:"name"`
		Code                   string `json:"code"`
		Split                  bool   `json:"split"`
		ExpectedOutput         string `json:"expected_output"`
		ExpectedOutputContains string `json:"expected_output_contains"`
		ExpectedStdout         string `json:"expected_stdout"`
		ExpectedStderr         string `json:"expected_stderr"`
	}
	ipyCasesJSON, err := os.ReadFile(os.Getenv("TEST_IPYTHON_CASES_PATH"))
	require.NoError(t, err, "reading ipython cases")
	require.NoError(t, json.Unmarshal(ipyCasesJSON, &ipyCases))
	require.GreaterOrEqual(t, len(ipyCases), 1, "ipython cases should not be empty")

	for _, tc := range ipyCases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := c.RunIPythonCell(ctx, space, createdSbx.Name, &v1.RunIPythonCellRequest{
				Code:        tc.Code,
				SplitOutput: tc.Split,
			})
			require.NoError(t, err, "Running IPython Cell")
			if tc.ExpectedOutputContains != "" {
				require.Contains(t, resp.Output, tc.ExpectedOutputContains, "output contains")
				require.Empty(t, tc.ExpectedOutput, "invalid assertion combo")
			} else {
				require.Equal(t, tc.ExpectedOutput, resp.Output, "output")
				require.Empty(t, tc.ExpectedOutputContains, "invalid assertion combo")
			}
			require.Equal(t, tc.ExpectedStdout, resp.Stdout, "stdout")
			require.Equal(t, tc.ExpectedStderr, resp.Stderr, "stderr")
		})
	}

	// Shell Command Tool //

	var shellCases []struct {
		Name           string `json:"name"`
		Command        string `json:"command"`
		Split          bool   `json:"split"`
		ExpectedOutput string `json:"expected_output"`
		ExpectedStdout string `json:"expected_stdout"`
		ExpectedStderr string `json:"expected_stderr"`
	}
	shellCasesJSON, err := os.ReadFile(os.Getenv("TEST_SHELL_CASES_PATH"))
	require.NoError(t, err, "reading shell cases")
	require.NoError(t, json.Unmarshal(shellCasesJSON, &shellCases))
	require.GreaterOrEqual(t, len(shellCases), 1, "shell cases should not be empty")

	for _, tc := range shellCases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := c.RunShellCommand(ctx, space, createdSbx.Name, &v1.RunShellCommandRequest{
				Command:     tc.Command,
				SplitOutput: tc.Split,
			})
			require.NoError(t, err, "Running shell command")
			require.Equal(t, tc.ExpectedOutput, resp.Output, "output")
			require.Equal(t, tc.ExpectedStdout, resp.Stdout, "stdout")
			require.Equal(t, tc.ExpectedStderr, resp.Stderr, "stderr")
		})
	}
}

func TestClientV1NoOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	c := clientv1.NewClient(cfg.SandboxAIBaseURL)
	require.NoError(t, c.CheckHealth(ctx))
}
