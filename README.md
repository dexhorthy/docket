docket
======

![report](https://goreportcard.com/badge/github.com/horthy/docket)

A lightweight cron server for running short-lived docker containers.

Install
--------

```
go get github.com/horthy/docket
```

Design
--------

### Assumptions
- Containers to be run are very short lived
- We'll run on a single host with an in-memory store. If the server is stopped
  all allocations are lost. (Although one could implement a persistent version of `AllocationSource`)

#### server

Core datatype is the `Allocation`:

```go
type Allocation struct {
    Name      string                        `json:"Name" `
    Cron      string                        `json:"Cron"`
    Container docker.CreateContainerOptions `json:"Container"`
    Logs      []interface{}                 `json:"Logs"`
    CronExpr  *cronexpr.Expression          `json:"-"`
}
```

Allocations can be scheduled by `POST`ing an `AllocationSpecification`


```go
type AllocationSpecification struct {
    Name      string                        `json:"Name" binding:"required"`
    Cron      string                        `json:"Cron" binding:"required"`
    Container docker.CreateContainerOptions `json:"Container" binding:"required"`
}
```

The Container options are directly from fsouza's Docker client, so any options there can be applied. 
For example, to echo some text every minute:

```sh
curl -X POST <server-ip-and-port> -d \
    '{
        "Name": "foo",
         "Cron":"* * * * * *",
        "Container":{
            "HostConfig":{
                "AutoRemove":true
            },
            "Config":{
                "Image":"busybox:latest",
                "Cmd":["echo", "hello world"]
            }
        }
    }' \
     -v -H 'Content-Type: application/json'
```


The server has 4 endpoints:

- `GET /` returns all allocations
- `GET /:name` returns the allocation named `:name`
- `DELETE /:name` deletes the allocation named `:name`
- `POST /` Creates a new allocation or updates an existing one

The server stores allocations in an implementation of `AllocationSource`.
By default, it uses `allocations.InMemory()`, which is backed by a go slice.

The server also runs a goroutine to check all the allocations every minute,
and pull+create+run any containers requested for that time in `Allocation.CronExpr`.


Commands
--------


### `server`

Start the server with `docket server`. The Port is not currently configurable.
The docker client used is built with `NewClientFromEnv`, so something like

```
DOCKER_HOST=tcp://192.168.100.1:2375 docket server
```

should work.

### `client`

Client commands all accept the flag `--host` for specifying a
server instance against which to run commands. Default is `http://localhost:3000`

#### `push`

Read a list of config from a yaml file. By default, looks for a file called `docket.yml` in the
current directory.

```sh
docket push
# or
docket push my_custom_docket_config.yml
```

`docket.yml` should contain a list of `AllocationSpecifications`:

```yaml
# example docket.yml
---
- name: foo
  cron: "* * * * * *"
  container:
      host_config:
          AutoRemove: true
      config:
          Image: busybox:latest
          Cmd:
              - echo
              - its foo
- name: bar
  cron: "* * * * * *"
  container:
      host_config:
          AutoRemove: true
      config:
          Image: openjdk:7-jre-alpine
          Cmd:
              - echo
              - its bar

- name: baz
  cron: "* * * * * *"
  container:
      host_config:
          AutoRemove: true
      config:
          Image: mattbailey/ch7-conductor
          Cmd:
              - echo
              - its baz
```


#### `list`

Once some allocations have been scheudled, they can be inspected with list.

Note that allocations include logs so whole classes of errors can be detected
and diagnosed using just the CLI client.

```sh
docket list
GET http://localhost:3000
[
  {
    "Container": {
      "Context": null,
      "NetworkingConfig": {
        "EndpointsConfig": null
      },
      "HostConfig": {
        "LogConfig": {},
        "RestartPolicy": {}
      },
      "Config": {
        "Entrypoint": null,
        "Image": "busybox:latest",
        "Cmd": [
          "echo",
          "its foo"
        ]
      },
      "Name": ""
    },
    "Cron": "* * * * * *",
    "Logs": [
      "2016-12-12 19:03:33.601558174 -0800 PST, [Pulled busybox latest foo]",
      "2016-12-12 19:03:33.659964749 -0800 PST, [created:  f8ecd244c7cc86656c4da1411ea1b669280326e716630641bed22d233b982d23]",
      "2016-12-12 19:03:33.788695323 -0800 PST, [started:  f8ecd244c7cc86656c4da1411ea1b669280326e716630641bed22d233b982d23]"
    ],
    "Name": "foo"
  },
  {
    "container": {
      "Context": null,
      "NetworkingConfig": {
        "EndpointsConfig": null
      },
      "HostConfig": {
        "LogConfig": {},
        "RestartPolicy": {}
      },
      "Config": {
        "Entrypoint": null,
        "Image": "openjdk:7-jre-alpine",
        "Cmd": [
          "/bin/sh",
          "-C",
          "echo its bar"
        ]
      },
      "Name": ""
    },
    "Cron": "* * * * * *",
    "Logs": [
      "2016-12-12 19:03:35.083605252 -0800 PST, [Pulled openjdk 7-jre-alpine bar]",
      "2016-12-12 19:03:35.144553286 -0800 PST, [created:  69b1fef49a30903e1edc549a22dfe30345a144e4bf6c5e4be8c93dd93af87360]",
      "2016-12-12 19:03:35.276230457 -0800 PST, [started:  69b1fef49a30903e1edc549a22dfe30345a144e4bf6c5e4be8c93dd93af87360]"
    ],
    "Name": "bar"
  },
  {
    "container": {
      "Context": null,
      "NetworkingConfig": {
        "EndpointsConfig": null
      },
      "HostConfig": {
        "LogConfig": {},
        "RestartPolicy": {}
      },
      "Config": {
        "Entrypoint": null,
        "Image": "mattbailey/dfksdalfaslkj",
        "Cmd": [
          "echo",
          "its baz"
        ]
      },
      "Name": ""
    },
    "Cron": "* * * * * *",
    "Logs": [
      "2016-12-12 19:03:36.906776241 -0800 PST, [Error: image mattbailey/golang-needs-generics not found]"
    ],
    "Name": "baz"
  }
]
```

#### `get`

A single allocation can be retrieved with `get`:

```
docket get foo
{
  "Container": {
    "Context": null,
    "NetworkingConfig": {
      "EndpointsConfig": null
    },
    "HostConfig": {
      "LogConfig": {},
      "RestartPolicy": {}
    },
    "Config": {
      "Entrypoint": null,
      "Image": "busybox:latest",
      "Cmd": [
        "echo",
        "its foo"
      ]
    },
    "Name": ""
  },
  "Cron": "* * * * * *",
  "Logs": [
    "2016-12-12 19:03:33.601558174 -0800 PST, [Pulled busybox latest foo]",
    "2016-12-12 19:03:33.659964749 -0800 PST, [created:  f8ecd244c7cc86656c4da1411ea1b669280326e716630641bed22d233b982d23]",
    "2016-12-12 19:03:33.788695323 -0800 PST, [started:  f8ecd244c7cc86656c4da1411ea1b669280326e716630641bed22d233b982d23]",
    "2016-12-12 19:04:33.569402505 -0800 PST, [Pulled busybox latest foo]",
    "2016-12-12 19:04:33.635877743 -0800 PST, [created:  87cb9223a5c94e4ab36bd4b053067f31f111a4d004af6e2515fca8c1529ba313]",
    "2016-12-12 19:04:33.764594194 -0800 PST, [started:  87cb9223a5c94e4ab36bd4b053067f31f111a4d004af6e2515fca8c1529ba313]",
    "2016-12-12 19:05:33.659443186 -0800 PST, [Pulled busybox latest foo]",
    "2016-12-12 19:05:33.73618851 -0800 PST, [created:  2beed47b2330e65538e2e3c6c8e9486a2f1ee9c84b8ea7603496d17d1bf9957b]",
    "2016-12-12 19:05:33.880550276 -0800 PST, [started:  2beed47b2330e65538e2e3c6c8e9486a2f1ee9c84b8ea7603496d17d1bf9957b]",
    "2016-12-12 19:06:33.614782569 -0800 PST, [Pulled busybox latest foo]",
    "2016-12-12 19:06:33.680620726 -0800 PST, [created:  b4b93b133282b7782a7beedb65f14b3ed36ccd14f0e2127b5f27955aa1bcc67d]",
    "2016-12-12 19:06:33.825758713 -0800 PST, [started:  b4b93b133282b7782a7beedb65f14b3ed36ccd14f0e2127b5f27955aa1bcc67d]",
    "2016-12-12 19:07:33.623988382 -0800 PST, [Pulled busybox latest foo]",
    "2016-12-12 19:07:33.688642129 -0800 PST, [created:  62c05b40e490166b517c1a709206ce31f91f99868339e506414bf86eff57c858]",
    "2016-12-12 19:07:33.818877136 -0800 PST, [started:  62c05b40e490166b517c1a709206ce31f91f99868339e506414bf86eff57c858]"
  ],
  "Name": "foo"
}
```


#### `delete`

We can delete an allocation with `delete`

```sh
$ docket delete foo
$ docket delete bar
$ docket delete baz
$ docket list
GET http://localhost:3000
[]
```
