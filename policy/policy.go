package policy

import (
	"go.podman.io/image/v5/signature"
)

// Context returns the default policy context.
func Context() (*signature.PolicyContext, error) {
	pol := &signature.Policy{
		Default: signature.PolicyRequirements{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	return signature.NewPolicyContext(pol)
}

// MustContext returns the default policy context or panics.
func MustContext() *signature.PolicyContext {
	context, err := Context()
	if err != nil {
		panic(err)
	}
	return context
}
