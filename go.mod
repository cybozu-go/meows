module github.com/cybozu-go/github-actions-controller

go 1.16

require (
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/go-logr/logr v0.3.0
	github.com/google/go-github/v33 v33.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/spf13/cobra v1.1.1
	go.uber.org/zap v1.15.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
)
