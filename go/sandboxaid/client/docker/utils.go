package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"math/rand"

	"github.com/docker/docker/api/types"
	v1 "github.com/substratusai/sandboxai/go/api/v1"
	sclient "github.com/substratusai/sandboxai/go/sandboxaid/client"
)

const (
	envHost       = "DOCKER_HOST"
	envAPIVersion = "DOCKER_API_VERSION"
	envCertPath   = "DOCKER_CERT_PATH"
	envTLSVerify  = "DOCKER_TLS_VERIFY"
)

// dockerContext represents the structure returned by `docker context inspect`.
type dockerContext struct {
	Name      string `json:"Name"`
	Endpoints struct {
		Docker struct {
			Host          string `json:"Host"`
			SkipTLSVerify bool   `json:"SkipTLSVerify"`
		} `json:"docker"`
	} `json:"Endpoints"`
	Storage struct {
		TLSPath string `json:"TLSPath"`
	} `json:"Storage"`
}

// setDockerEnvFromContext inspects the current Docker CLI context and sets
// the following env variables if they are not set:
//
// - DOCKER_HOST
// - DOCKER_API_VERSION
// - DOCKER_CERT_PATH
// - DOCKER_TLS_VERIFY
//
// NOTE: You would think that this would be a part of the Docker client library, but it's not,
// it is a part of the cli codebase.
func setDockerEnvFromContextIfNotSet() error {
	// Check if env variables are already set.
	_, hostSet := os.LookupEnv(envHost)
	_, apiVersionSet := os.LookupEnv(envAPIVersion)
	_, certPathSet := os.LookupEnv(envCertPath)
	_, tlsVerifySet := os.LookupEnv(envTLSVerify)

	// If all are set, we do nothing.
	if hostSet && apiVersionSet && certPathSet && tlsVerifySet {
		return nil
	}

	// DOCKER_API_VERSION
	if err := setDockerAPIVersionFromEngineIfNotSet(); err != nil {
		return err
	}

	// 1) Get the name of the current context
	out, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return fmt.Errorf("failed to get current docker context: %w", err)
	}
	currentContext := strings.TrimSpace(string(out))

	// 2) Inspect the current context
	out, err = exec.Command("docker", "context", "inspect", currentContext).Output()
	if err != nil {
		return fmt.Errorf("failed to inspect docker context: %w", err)
	}

	// 3) Parse the JSON output (an array of contexts).
	var contexts []dockerContext
	if err := json.Unmarshal(bytes.TrimSpace(out), &contexts); err != nil {
		return fmt.Errorf("failed to parse context JSON: %w", err)
	}
	if len(contexts) < 1 {
		return fmt.Errorf("no context data returned for context %q", currentContext)
	}
	ctx := contexts[0]

	// DOCKER_HOST
	if !hostSet && ctx.Endpoints.Docker.Host != "" {
		if err := os.Setenv(envHost, ctx.Endpoints.Docker.Host); err != nil {
			return fmt.Errorf("failed to set %s: %w", envHost, err)
		}
	}

	// DOCKER_CERT_PATH
	if !certPathSet && ctx.Storage.TLSPath != "" {
		// Only set if this path contains certs
		certFilePath := filepath.Join(ctx.Storage.TLSPath, "ca.pem")
		if _, err := os.Stat(certFilePath); err == nil {
			if err := os.Setenv(envCertPath, ctx.Storage.TLSPath); err != nil {
				return fmt.Errorf("failed to set %s: %w", envCertPath, err)
			}
		}
	}

	// DOCKER_TLS_VERIFY
	if !tlsVerifySet {
		// Docker sets SkipTLSVerify == true => insecure => TLS_VERIFY=0
		// If SkipTLSVerify == false, we want to verify => TLS_VERIFY=1
		var val string
		if ctx.Endpoints.Docker.SkipTLSVerify {
			val = "0"
		} else {
			val = "1"
		}
		if err := os.Setenv(envTLSVerify, val); err != nil {
			return fmt.Errorf("failed to set %s: %w", envTLSVerify, err)
		}
	}

	return nil
}

func setDockerAPIVersionFromEngineIfNotSet() error {
	// First check if DOCKER_API_VERSION is already set
	if _, ok := os.LookupEnv(envAPIVersion); ok {
		return nil
	}

	// Run `docker version` in "Go template" mode to output only the server's API version
	cmd := exec.Command("docker", "version", "--format={{.Server.APIVersion}}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Docker server API version: %w: %s", err, string(out))
	}

	apiVersion := strings.TrimSpace(string(out))
	if apiVersion == "" {
		// if for some reason the output was empty, handle accordingly
		return fmt.Errorf("could not retrieve Docker server API version (output was empty)")
	}

	// Set the environment variable
	if err := os.Setenv(envAPIVersion, apiVersion); err != nil {
		return fmt.Errorf("failed to set %s: %w", envAPIVersion, err)
	}

	return nil
}

func containerName(space, name string) string {
	return fmt.Sprintf("%s.%s", space, name)
}

func containerJSONToSandbox(c types.ContainerJSON) (*sclient.Sandbox, error) {
	var env map[string]string
	if len(c.Config.Env) > 0 {
		for _, kv := range c.Config.Env {
			if env == nil {
				env = make(map[string]string)
			}
			key, val := parseEnvKeyVal(kv)
			if key != "" {
				env[key] = val
			}
		}
	}

	boxHostPort, err := getBoxHostPort(c)
	if err != nil {
		return nil, fmt.Errorf("container %q: getting box host port: %w", c.Name, err)
	}

	name := c.Config.Labels[labelKeyName]

	return &sclient.Sandbox{
		Sandbox: &v1.Sandbox{
			Name: name,
			UID:  c.ID,
			Spec: v1.SandboxSpec{
				Image: c.Config.Image,
				Env:   env,
			},
		},
		BoxHostPort: boxHostPort,
	}, nil
}

func getBoxHostPort(dockerContainer types.ContainerJSON) (int, error) {
	boxHostPortStr := dockerContainer.NetworkSettings.Ports["8000/tcp"][0].HostPort
	boxHostPort, err := strconv.Atoi(boxHostPortStr)
	if err != nil {
		return 0, fmt.Errorf("converting docker host port string %q to int: %w", boxHostPortStr, err)
	}
	return boxHostPort, nil
}
func parseEnvKeyVal(s string) (string, string) {
	key, val, ok := strings.Cut(s, "=")
	if ok {
		return key, val
	}
	return "", ""
}

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func generateRandomName() string {
	const length = 20
	const charset = "abcdefghijklmnopqrstuvwxyz1234567890"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
