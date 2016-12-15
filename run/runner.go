package run

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/horthy/docket/allocations"
	"log"
)

type AllocationRunner interface {
	RunAllocation(alloc *allocations.Allocation)
}

type FsouzaAllocationRunner struct {
	store  allocations.AllocationStore
	client *docker.Client
}

func NewFsouza(
	client *docker.Client,
	store allocations.AllocationStore,
) *FsouzaAllocationRunner {
	return &FsouzaAllocationRunner{
		client: client,
		store:  store,
	}
}

func (runner *FsouzaAllocationRunner) RunAllocation(alloc *allocations.Allocation) {
	log.Printf("Creating container for allocation %v with cron %v", alloc.Name, alloc.Cron)

	// pull image -- might want to this on allocation creation so we can bail
	// if the image doesn't exist, but leaving it here for now
	err := runner.pullImage(alloc)
	if err != nil {
		return
	}

	container, err := runner.createContainer(alloc)
	if err != nil {
		return
	}

	runner.startContainer(alloc, container)
}

func (runner *FsouzaAllocationRunner) pullImage(alloc *allocations.Allocation) error {
	repo, tag := docker.ParseRepositoryTag(alloc.Container.Config.Image)
	opts := docker.PullImageOptions{
		Repository: repo,
		Tag:        tag,
	}

	log.Printf("Pulling %v:%v for %v", repo, tag, alloc.Name)
	err := runner.client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		log.Printf("Failed to pull image for %v, error was %v", alloc.Name, err)
		runner.store.Log(alloc, err)
		return err
	}
	log.Printf("Pulled %v:%v for allocation %v", repo, tag, alloc.Name)
	runner.store.Log(alloc, "Pulled", repo, tag, alloc.Name)
	return nil
}

func (runner *FsouzaAllocationRunner) createContainer(alloc *allocations.Allocation) (*docker.Container, error) {
	//create container
	container, err := runner.client.CreateContainer(alloc.Container.ToOptions())
	if err != nil {
		log.Printf("Failed to create container for %v, error was %v", alloc.Name, err)
		runner.store.Log(alloc, err)
		return nil, err
	}

	log.Printf("created: %v %v", container.Name, container.ID)
	runner.store.Log(alloc, "created:", container.Name, container.ID)
	return container, nil
}

func (runner *FsouzaAllocationRunner) startContainer(alloc *allocations.Allocation, container *docker.Container) {
	// start
	err := runner.client.StartContainer(container.ID, alloc.Container.HostConfig)
	if err != nil {
		log.Printf("Failed to start container for %v, error was %v", alloc.Name, err)
		runner.store.Log(alloc, err)
		runner.client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
		log.Printf("tried to remove container for %v, error was %v", alloc.Name, err)
		runner.store.Log(alloc, "removed container because", err)
		return
	}
	log.Printf("started: %v %v", container.Name, container.ID)
	runner.store.Log(alloc, "started:", container.Name, container.ID)

}
