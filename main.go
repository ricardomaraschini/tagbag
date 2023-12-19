package main

import (
	"context"
	"tagbag/storage"
	"tagbag/versioner"

	imgcopy "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func main() {
	pol := &signature.Policy{
		Default: signature.PolicyRequirements{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	polctx, err := signature.NewPolicyContext(pol)
	if err != nil {
		panic(err)
	}
	versioner, err := versioner.NewGit("/tmp/test")
	if err != nil {
		panic(err)
	}
	storage := storage.New("/tmp/test", versioner)
	if err := storage.Image("ricardomaraschini/nginx:latest"); err != nil {
		panic(err)
	}
	src := "docker://docker.io/ricardomaraschini/test:latest"
	srcref, err := alltransports.ParseImageName(src)
	if err != nil {
		panic(err)
	}
	_, err = imgcopy.Image(
		context.Background(),
		polctx,
		storage,
		srcref,
		&imgcopy.Options{
			SourceCtx:      &types.SystemContext{},
			DestinationCtx: &types.SystemContext{},
		},
	)
	if err != nil {
		panic(err)
	}
	if err := versioner.Snapshot("v1.0.0"); err != nil {
		panic(err)
	}

	////////////////
	err = storage.Image("alpine")
	if err != nil {
		panic(err)
	}
	src = "docker://docker.io/library/alpine:latest"
	srcref, err = alltransports.ParseImageName(src)
	if err != nil {
		panic(err)
	}
	_, err = imgcopy.Image(
		context.Background(),
		polctx,
		storage,
		srcref,
		&imgcopy.Options{
			SourceCtx:      &types.SystemContext{},
			DestinationCtx: &types.SystemContext{},
		},
	)
	if err != nil {
		panic(err)
	}
	if err := versioner.Snapshot("v2.0.0"); err != nil {
		panic(err)
	}

	////////////////
	err = storage.Image("urussanga")
	if err != nil {
		panic(err)
	}
	src = "docker://docker.io/ricardomaraschini/test:latest"
	srcref, err = alltransports.ParseImageName(src)
	if err != nil {
		panic(err)
	}
	_, err = imgcopy.Image(
		context.Background(),
		polctx,
		storage,
		srcref,
		&imgcopy.Options{
			SourceCtx:      &types.SystemContext{},
			DestinationCtx: &types.SystemContext{},
		},
	)
	if err != nil {
		panic(err)
	}
	if err := versioner.Snapshot("v3.0.0"); err != nil {
		panic(err)
	}

	////////////////
	err = storage.Image("nginx:latest")
	if err != nil {
		panic(err)
	}
	src = "docker://docker.io/library/nginx:latest"
	srcref, err = alltransports.ParseImageName(src)
	if err != nil {
		panic(err)
	}
	_, err = imgcopy.Image(
		context.Background(),
		polctx,
		storage,
		srcref,
		&imgcopy.Options{
			SourceCtx:      &types.SystemContext{},
			DestinationCtx: &types.SystemContext{},
		},
	)
	if err != nil {
		panic(err)
	}
	if err := versioner.Snapshot("v4.0.0"); err != nil {
		panic(err)
	}
}
