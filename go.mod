module github.com/dhouti/sops-converter

go 1.13

require (
	github.com/aws/aws-sdk-go v1.23.13
	github.com/go-logr/logr v0.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	go.mozilla.org/sops/v3 v3.6.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
