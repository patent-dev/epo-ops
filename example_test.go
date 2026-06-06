package epo_ops_test

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
)

func ExampleNewClient() {
	client, err := ops.NewClient(&ops.Config{
		ConsumerKey:    "your-consumer-key",
		ConsumerSecret: "your-consumer-secret",
	})
	if err != nil {
		panic(err)
	}
	// With a valid client, call e.g. client.GetBiblio(ctx, "publication", "docdb", "EP.1000000.B1").
	fmt.Println(client != nil)
	// Output: true
}
