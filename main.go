// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version = "dev"
	commit  = "none" // set via -X main.commit by GoReleaser

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

// Format Terraform code for use in documentation.
// If you do not have Terraform installed, you can remove the formatting command, but it is suggested
// to ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Generate documentation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name jira
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate -provider-name jira

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/devops-wiz/jira",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(fmt.Sprintf("%s+commit.%s", version, commit)), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
