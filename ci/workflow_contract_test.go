package ci

import (
	"os"
	"strings"
	"testing"
)

const sharedPublisherSHA = "a86300fb8020d0f7141bb9f833d89b5dbd7aa4d7"

func TestImagePublicationWorkflowContract(t *testing.T) {
	workflow, err := os.ReadFile("../.github/workflows/build-push.yml")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(workflow)

	required := []string{
		"pull_request:",
		"if: github.event_name == 'pull_request'",
		"if: github.ref == 'refs/heads/main' || (github.event_name == 'push' && startsWith(github.ref, 'refs/tags/'))",
		"Build native image without credentials",
		"libops/.github/.github/workflows/build-push.yaml@" + sharedPublisherSHA,
		"ref: ${{ github.sha }}",
		"expected-main-sha: ${{ github.ref == 'refs/heads/main' && github.sha || '' }}",
		"sign: true",
		"certificate-identity: https://github.com/libops/.github/.github/workflows/build-push.yaml@" + sharedPublisherSHA,
		"packages: write",
		"id-token: write",
	}
	for _, value := range required {
		if !strings.Contains(contents, value) {
			t.Errorf("image workflow must contain %q", value)
		}
	}

	forbidden := []string{
		"build-push.yaml@main",
		"build-push-ghcr.yaml",
		"secrets: inherit",
		"docker-registry:",
		"additional-gar-registry:",
	}
	for _, value := range forbidden {
		if strings.Contains(contents, value) {
			t.Errorf("image workflow must not contain %q", value)
		}
	}
}
