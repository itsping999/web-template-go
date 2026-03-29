package discoveryx

import (
	"errors"
	"os"
	"strings"

	kuberegistry "github.com/go-kratos/kratos/contrib/registry/kubernetes/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewKubernetesRegistrar(cfg Config, logger log.Logger) (registry.Registrar, func(), error) {
	kcfg := cfg.Kubernetes
	if logger == nil {
		logger = log.NewStdLogger(os.Stdout)
	}
	helper := log.NewHelper(log.With(logger, "module", "discoveryx/kubernetes"))
	if !kcfg.Enabled {
		helper.Info("kubernetes registry disabled, skip initialization")
		return nil, func() {}, nil
	}

	restConfig, err := buildRestConfig(kcfg)
	if err != nil {
		return nil, nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	namespace := strings.TrimSpace(kcfg.Namespace)
	if namespace == "" {
		namespace = kuberegistry.GetNamespace()
	}
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	reg := kuberegistry.NewRegistry(clientSet, namespace)
	reg.Start()
	helper.Infow("msg", "kubernetes registry connected", "namespace", namespace, "mode", normalizeMode(kcfg.Mode))

	cleanup := func() {
		reg.Close()
		helper.Info("kubernetes registry closed")
	}
	return reg, cleanup, nil
}

func normalizeMode(mode string) string {
	v := strings.TrimSpace(mode)
	if v == "" {
		return "in_cluster"
	}
	return v
}

func buildRestConfig(cfg KubernetesConfig) (*rest.Config, error) {
	mode := normalizeMode(cfg.Mode)
	switch mode {
	case "in_cluster":
		return buildInClusterConfig(cfg)
	case "kubeconfig":
		return buildKubeconfig(cfg)
	default:
		return nil, errors.New("kubernetes mode is invalid")
	}
}

func buildInClusterConfig(cfg KubernetesConfig) (*rest.Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	applyOverrides(restConfig, cfg)
	return restConfig, nil
}

func buildKubeconfig(cfg KubernetesConfig) (*rest.Config, error) {
	if strings.TrimSpace(cfg.Kubeconfig) == "" {
		return nil, errors.New("kubernetes kubeconfig is empty")
	}
	restConfig, err := clientcmd.BuildConfigFromFlags(strings.TrimSpace(cfg.APIServer), strings.TrimSpace(cfg.Kubeconfig))
	if err != nil {
		return nil, err
	}
	applyOverrides(restConfig, cfg)
	return restConfig, nil
}

func applyOverrides(restConfig *rest.Config, cfg KubernetesConfig) {
	if restConfig == nil {
		return
	}
	if strings.TrimSpace(cfg.APIServer) != "" {
		restConfig.Host = strings.TrimSpace(cfg.APIServer)
	}
	if strings.TrimSpace(cfg.Token) != "" {
		restConfig.BearerToken = strings.TrimSpace(cfg.Token)
	}
	if cfg.Timeout > 0 {
		restConfig.Timeout = cfg.Timeout
	}
	if cfg.InsecureSkipVerify {
		restConfig.TLSClientConfig.Insecure = true
	}
}
