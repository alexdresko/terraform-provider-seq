package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/example/terraform-provider-seq/internal/provider"
)

var version = "dev"

// terraform-provider-seq entrypoint.
func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "start provider in debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/example/seq",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
