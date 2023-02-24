

***This was a POC collector for [Netapp's harvest](https://github.com/NetApp/harvest) and is probably not compatible with its current version anymore***

# Github

Collects some basic stats about a Github repository, such as number of downloads, open issues, traffic, and number of lines of each file.

This collector is mainly here to experiment with Harvest module architecture. I wanted to have a proof-of-concept that you can write a Harvest collector without touching the code base of main components.


## Installing

1. Clone this repo:

```sh
cd /opt
git clone --depth 1 https://github.com/vgratian/GithubCollector
ln -s /opt/GitHubCollector/cmd/collector/ /opt/harvest/cmd/collectors/github
ln -s /opt/GitHubCollector/conf/github/ /opt/harvest/conf/github
```

2. Make harvest import this module:
```
cd /opt/harvest
sed -i 's|"fmt"|"fmt"\n\t_ "goharvest2/cmd/collectors/github"|' cmd/poller/poller.go
```

2. Install dependencies and re-build Harvest:

```sh
cd /opt/harvest
go mod tidy
go mod vendor
make build
```

*Congrats you installed a new collector!*


## Parameters

| parameter  | type     | description                                      | 
|------------|----------|--------------------------------------------------|
| `addr`     | string       | Address of the repository (e.g. `https://github.com/NetApp/harvest`)   | 
| `password` | string   | A Github [personal access token](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token)     |

Next, to run the collector, define a new poller in your `harvest.yml` with these two parameters. Example: 

```yaml


Pollers:
  github:
    addr: https://github.com/NetApp/harvest
    password: ghp_sdffe%$^v6ub7b67RCERDWR$@
    collectors:
      - Github
    exporters:
      - prom

```

## Subtemplate and Metrics

The Collector will run two polls:
* `files`: provides to metrics: `repo_size_bytes` and `repo_size_lines`
* `data`: provides metrics from the APIs defined in the `counters` section of the [conf/github/default.yaml](conf/github/default.yaml).
