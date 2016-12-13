// this package provides an implementation of AllocationSource
// that works by making HTTP calls to a server instance
//
// There is a lot of duplication between the methods here,
// and it would be nice to
package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/horthy/docket/allocations"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	baseUrl string
}

func NewClient(baseUrl string) *Client {
	return &Client{
		baseUrl: baseUrl,
	}
}

func (c *Client) List() (allocations.Allocations, error) {
	fmt.Fprint(os.Stderr, color.BlueString("GET %v\n", c.baseUrl))

	result, err := c.execute(
		func() (*http.Response, error) { return http.Get(c.baseUrl) },
		&allocations.Allocations{},
	)

	if err != nil {
		return nil, err
	}

	cast, ok := result.(*allocations.Allocations)
	if !ok {
		return nil, errors.New("error casting response to *allocations.Allocations")
	}

	return *cast, nil
}

func (c *Client) Get(name string) (*allocations.Allocation, error) {
	url := strings.Join([]string{c.baseUrl, name}, "/")
	fmt.Fprint(os.Stderr, color.BlueString("GET %v\n", url))
	result, err := c.execute(
		func() (*http.Response, error) { return http.Get(url) },
		&allocations.Allocation{},
	)

	if err != nil {
		return nil, err
	}

	cast, ok := result.(*allocations.Allocation)
	if !ok {
		return nil, errors.New("error casting response to *allocations.Allocation")
	}

	return cast, nil
}

func (c *Client) CreateOrUpdate(newAllocation *allocations.AllocationSpecification) (bool, error) {

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(newAllocation)
	if err != nil {
		return false, err
	}

	fmt.Fprint(os.Stderr, color.BlueString("POST %v", c.baseUrl))

	pretty, err := json.MarshalIndent(newAllocation, "", "    ")
	fmt.Println(string(pretty))

	result, err := c.execute(
		func() (*http.Response, error) { return http.Post(c.baseUrl, "application/json", buffer) },
		&map[string]bool{},
	)

	if err != nil {
		return false, err
	}

	cast, ok := result.(*map[string]bool)
	if !ok {
		return false, errors.New("error casting response to map[string]bool")
	}

	return (*cast)["created"], nil
}

func (c *Client) Delete(name string) error {
	url := strings.Join([]string{c.baseUrl, name}, "/")
	fmt.Fprint(os.Stderr, color.BlueString("DELETE %v", url))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = c.execute(
		func() (*http.Response, error) { return http.DefaultClient.Do(req) },
		&struct{}{},
	)

	if err != nil {
		return err
	}

	return nil
}

// run an http call and marshall the result into a target
// There may be a library to do this, and I'm not even sure trading casting
// for code dupe is even worth it.
func (c *Client) execute(call func() (*http.Response, error), target interface{}) (interface{}, error) {
	resp, err := call()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Server responded with status %v body %v", resp.Status, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, target)
	if err != nil {
		return nil, err
	}

	return target, nil
}
