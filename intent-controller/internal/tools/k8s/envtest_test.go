package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func TestMain(m *testing.M) {
	// Setup envtest
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	basePath := filepath.Join("..", "..", "..", "bin", "k8s")
	if entries, err := os.ReadDir(basePath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				testEnv.BinaryAssetsDirectory = filepath.Join(basePath, entry.Name())
				break
			}
		}
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start envtest: %v\n", err)
		os.Exit(1)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		fmt.Printf("failed to create client: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	err = testEnv.Stop()
	if err != nil {
		fmt.Printf("failed to stop envtest: %v\n", err)
	}

	os.Exit(code)
}

func TestEnvtestApply(t *testing.T) {
	ctx := context.Background()

	input := ApplyInput{
		Resource: "configmaps",
		Manifest: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-envtest-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}

	out, err := handleApply(ctx, k8sClient, input)
	if err != nil {
		t.Fatalf("handleApply failed: %v", err)
	}
	if out.Message == "" {
		t.Errorf("expected success message, got empty")
	}

	// Verify it was actually created
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"})
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-envtest-cm"}, u)
	if err != nil {
		t.Fatalf("failed to fetch created configmap: %v", err)
	}
}
