package e2e_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/sethvargo/go-envconfig"
)

var ctx = context.Background()
var cfg struct {
	SandboxAIBaseURL string `env:"SANDBOXAI_BASE_URL"`
	BoxImage         string `env:"BOX_IMAGE"`
}

func TestMain(m *testing.M) {
	envconfig.MustProcess(ctx, &cfg)
	log.Printf("sandboxaid base URL: %q", cfg.SandboxAIBaseURL)
	log.Printf("Box image: %q", cfg.BoxImage)
	os.Exit(m.Run())
}
