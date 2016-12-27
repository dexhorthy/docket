package server

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/fsouza/go-dockerclient"
	"github.com/horthy/docket/allocations"
	"github.com/horthy/docket/run"
	"github.com/horthy/docket/store"
	"github.com/spf13/viper"
	"log"
	"time"
)

func initServerConfig() *viper.Viper {
	v := viper.New()

	v.SetConfigFile("docket")
	v.AddConfigPath("/etc/docket")

	v.SetDefault("host", "localhost")
	v.SetDefault("port", "3000")
	v.SetDefault("store", "InMemory")

	v.SetEnvPrefix("docket")
	v.BindEnv("port", "store", "host")

	return v
}

func Start() {

	config := initServerConfig()

	factory := store.StoreImpls[config.GetString("store")]

	if factory == nil {
		log.Fatalf("No implementation found for backend store: %v", config.Get("store"))
	}

	allocationStore, err := factory.Create(config)
	if err != nil {
		log.Fatalf("Couldn't initialize allocation store: %v", err)
	}

	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(func(c martini.Context) {
		c.MapTo(allocationStore, (*store.AllocationStore)(nil))
	})

	m.Get("/", handleGet)
	m.Get("/:name", handleGetAllocation)
	m.Delete("/:name", handleDeleteAllocation)
	m.Post("/", binding.Bind(allocations.AllocationSpecification{}), handlePost)

	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// TODO
	// using ticker feels kinda janky -- even if we continue to maintain our own collection
	// of Allocations (as opposed to just registering each request using some cron package),
	// we can still use a cron library to manage scheduling our checks

	ticker := time.NewTicker(1 * time.Minute)
	runner := run.NewFsouza(client, allocationStore)
	go func() {
		for range ticker.C {
			RunAnyScheduledContainers(runner, allocationStore)
		}
	}()

	log.Printf("%v", config)

	m.RunOnAddr(config.GetString("host") + ":" + config.GetString("port"))

	// TODO/nice to have: watch docker event stream, add exit codes to Allocation Logs
}

func handlePost(
	allocation allocations.AllocationSpecification,
	allocationStore store.AllocationStore,
	r render.Render,
) {

	allocation.ProvisionDefaults()
	pretty, err := json.MarshalIndent(allocation, "", "    ")
	log.Printf("Received new allocation %v", string(pretty))

	created, err := allocationStore.CreateOrUpdate(&allocation)
	if err != nil {
		log.Printf("Failed to store allocation %v, error was %v", pretty, err)
		r.JSON(500, err)
	} else {
		log.Printf("Stored allocation %v", allocation.Name)
		r.JSON(200, map[string]bool{"created": created})
	}
}

func handleGet(allocationStore store.AllocationStore, r render.Render) {
	list, err := allocationStore.List()
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, list)
	}
}

func handleGetAllocation(allocationStore store.AllocationStore, r render.Render, params martini.Params) {
	allocation, err := allocationStore.Get(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, allocation)
	}
}

func handleDeleteAllocation(allocationStore store.AllocationStore, r render.Render, params martini.Params) {

	err := allocationStore.Delete(params["name"])
	if err != nil {
		r.JSON(500, err)
	} else {
		r.JSON(200, map[string]bool{"deleted": true})
	}
}

// Check the list of allocations, and create+start any
// containers that are scheduled at the time the method is called
func RunAnyScheduledContainers(runner run.AllocationRunner, allocationStore store.AllocationStore) {
	allAllocations, err := allocationStore.List()

	if err != nil {
		log.Printf("Couldn't get list of allocations, error was %v", err)
		return
	}

	for _, alloc := range allAllocations {
		if alloc.ShouldRunAt(time.Now()) {
			runner.RunAllocation(alloc)
		}
	}
}
