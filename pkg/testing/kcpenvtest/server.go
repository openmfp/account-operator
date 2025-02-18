package kcpenvtest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/openmfp/golang-commons/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kcpapiv1alpha "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcptenancyv1alpha "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
)

const (
	kcpEnvStartTimeout        = "KCP_SERVER_START_TIMEOUT"
	kcpEnvStopTimeout         = "KCP_SERVER_STOP_TIMEOUT"
	defaulKCPServerTimeout    = 20 * time.Second
	defaultKCPServerTimeout   = 20 * time.Second
	kcpAdminKubeconfigPath    = ".kcp/admin.kubeconfig"
	kcpRootNamespaceServerUrl = "https://localhost:6443/clusters/root"
	dirOrderPattern           = `^[0-9]*-(.*)$`
)

type Environment struct {
	kcpServer *KCPServer

	Scheme *runtime.Scheme

	ControlPlaneStartTimeout time.Duration

	ControlPlaneStopTimeout time.Duration

	Config *rest.Config

	log *logger.Logger

	RelativeSetupDirectory string

	PathToRoot             string
	RelativeAssetDirectory string

	ProviderWorkspace string
	APIExportName     string
}

func NewEnvironment(apiExportName string, providerWorkspaceName string, pathToRoot string, relativeAssetDirectory string, relativeSetupDirectory string, log *logger.Logger) *Environment {
	kcpBinary := filepath.Join(relativeAssetDirectory, "kcp")
	kcpServ := NewKCPServer(pathToRoot, kcpBinary, pathToRoot, log)
	//kcpServ.Out = os.Stdout
	//kcpServ.Err = os.Stderr
	return &Environment{
		log:                    log,
		kcpServer:              kcpServ,
		APIExportName:          apiExportName,
		ProviderWorkspace:      providerWorkspaceName,
		RelativeSetupDirectory: relativeSetupDirectory,
		RelativeAssetDirectory: relativeAssetDirectory,
		PathToRoot:             pathToRoot,
	}
}

func (te *Environment) Start() (*rest.Config, string, error) {
	// ensure clean .kcp directory
	te.cleanDir()

	if err := te.defaultTimeouts(); err != nil {
		return nil, "", fmt.Errorf("failed to default controlplane timeouts: %w", err)
	}
	//te.kcpServer.StartTimeout = te.ControlPlaneStartTimeout
	//te.kcpServer.StopTimeout = te.ControlPlaneStopTimeout

	te.log.Info().Msg("starting control plane")
	if err := te.kcpServer.Start(); err != nil {
		return nil, "", fmt.Errorf("unable to start control plane itself: %w", err)
	}

	if te.Scheme == nil {
		te.Scheme = scheme.Scheme
		utilruntime.Must(kcpapiv1alpha.AddToScheme(te.Scheme))
		utilruntime.Must(kcptenancyv1alpha.AddToScheme(te.Scheme))
	}
	//// wait for default namespace to actually be created and seen as available to the apiserver
	if err := te.waitForDefaultNamespace(); err != nil {
		return nil, "", fmt.Errorf("default namespace didn't register within deadline: %w", err)
	}

	kubectlPath := filepath.Join(te.PathToRoot, ".kcp", "admin.kubeconfig")
	var err error
	te.Config, err = clientcmd.BuildConfigFromFlags("", kubectlPath)
	if err != nil {
		return nil, "", err
	}
	te.Config.Host = kcpRootNamespaceServerUrl
	te.Config.QPS = 1000.0
	te.Config.Burst = 2000.0
	if te.RelativeSetupDirectory != "" {
		// Apply all yaml files in the setup directory
		setupDirectory := filepath.Join(te.PathToRoot, te.RelativeSetupDirectory)
		kubeconfigPath := filepath.Join(te.PathToRoot, kcpAdminKubeconfigPath)
		err := te.ApplyYAML(kubeconfigPath, te.Config, setupDirectory, kcpRootNamespaceServerUrl)
		if err != nil {
			return nil, "", err
		}
	}

	// Select api export
	providerServerUrl := fmt.Sprintf("%s:%s", te.Config.Host, te.ProviderWorkspace)
	te.Config.Host = providerServerUrl
	cs, err := client.New(te.Config, client.Options{})
	if err != nil {
		return nil, "", fmt.Errorf("unable to create client: %w", err)
	}

	apiExport := kcpapiv1alpha.APIExport{}
	err = cs.Get(context.Background(), types.NamespacedName{Name: te.APIExportName}, &apiExport)
	if err != nil {
		return nil, "", err
	}

	if len(apiExport.Status.VirtualWorkspaces) == 0 {
		return nil, "", fmt.Errorf("no virtual workspaces found")
	}
	return te.Config, apiExport.Status.VirtualWorkspaces[0].URL, nil
}

func (te *Environment) Stop() error {
	defer te.cleanDir()
	return te.kcpServer.Stop()
}

func (te *Environment) cleanDir() error {
	kcpPath := filepath.Join(te.PathToRoot, ".kcp")
	return os.RemoveAll(kcpPath)
}

func (te *Environment) waitForDefaultNamespace() error {
	kubectlPath := filepath.Join(te.PathToRoot, ".kcp", "admin.kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubectlPath)
	if err != nil {
		return err
	}
	cs, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	// It shouldn't take longer than 5s for the default namespace to be brought up in etcd
	return wait.PollUntilContextTimeout(context.TODO(), time.Millisecond*50, time.Second*10, true, func(ctx context.Context) (bool, error) {
		te.log.Info().Msg("waiting for default namespace")
		if err = cs.Get(ctx, types.NamespacedName{Name: "default"}, &corev1.Namespace{}); err != nil {
			te.log.Info().Msg("namespace not found")
			return false, nil //nolint:nilerr
		}
		return true, nil
	})
}

func (te *Environment) waitForWorkspace(client client.Client, name string) error {
	// It shouldn't take longer than 5s for the default namespace to be brought up in etcd
	return wait.PollUntilContextTimeout(context.TODO(), time.Millisecond*50, time.Second*5, true, func(ctx context.Context) (bool, error) {
		ws := &kcptenancyv1alpha.Workspace{}
		if err := client.Get(ctx, types.NamespacedName{Name: name}, ws); err != nil {
			return false, nil //nolint:nilerr
		}
		return ws.Status.Phase == "Ready", nil
	})
}

func (te *Environment) defaultTimeouts() error {
	var err error
	if te.ControlPlaneStartTimeout == 0 {
		if envVal := os.Getenv(kcpEnvStartTimeout); envVal != "" {
			te.ControlPlaneStartTimeout, err = time.ParseDuration(envVal)
			if err != nil {
				return err
			}
		} else {
			te.kcpServer.StartTimeout = defaulKCPServerTimeout
		}
	}

	if te.ControlPlaneStopTimeout == 0 {
		if envVal := os.Getenv(kcpEnvStopTimeout); envVal != "" {
			te.ControlPlaneStopTimeout, err = time.ParseDuration(envVal)
			if err != nil {
				return err
			}
		} else {
			te.ControlPlaneStopTimeout = defaultKCPServerTimeout
		}
	}
	return nil
}

func (te *Environment) ApplyYAML(pathToRootConfig string, config *rest.Config, dir string, serverUrl string) error {
	cs, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	// list directory
	err = te.runKubectlCommand(pathToRootConfig, serverUrl, fmt.Sprintf("apply -f %s", dir))
	if err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			fileName := file.Name()
			// check if dir starts with `[0-9]*-`
			re := regexp.MustCompile(dirOrderPattern)

			if re.Match([]byte(fileName)) {
				match := re.FindStringSubmatch(fileName)
				fileName = match[1]
			}
			err := te.waitForWorkspace(cs, fileName)
			if err != nil {
				return err
			}
			newServerUrl := fmt.Sprintf("%s:%s", serverUrl, fileName)
			wsConfig := rest.CopyConfig(config)
			wsConfig.Host = newServerUrl
			dir := filepath.Join(dir, file.Name())
			err = te.ApplyYAML(pathToRootConfig, wsConfig, dir, newServerUrl)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (te *Environment) runKubectlCommand(kubeconfig string, server string, command string) error {
	splitCommand := strings.Split(command, " ")
	args := []string{fmt.Sprintf("--kubeconfig=%s", kubeconfig), fmt.Sprintf("--server=%s", server)}
	args = append(args, splitCommand...)
	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
