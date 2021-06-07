

# Github

Collects basic stats about a Github repo. This collector is mainly here to experiment with Harvest module architecture.

## Target System
Github

## Requirements
Harvest with the [newly proposed module architecture](https://github.com/NetApp/harvest/tree/module-arch-improvement).

Install collector:

- add URL of this repository `/opt/harvest/modules.txt`
- copy `conf/github/` to `/opt/harvest/conf`
- run `make modules` and `make build`

## Parameters

| parameter              | type         | description                                      | default                |
|------------------------|--------------|--------------------------------------------------|------------------------|
| `add`                  | string       | URL of Github (`https://github.com`)             |                        |
| `repos`     | list of strings | list of Github repositories with format `OWNER/REPO` (e.g. `NetApp/harvest`)  |   |

## Metrics

Collects metrics using Guthub's REST APIs (list can be easily extended):

| metric             | type                       | unit          | description                                              |
|--------------------|----------------------------|---------------|----------------------------------------------------------|
| `size`             | counter, `uint64`          | byte          | size of the repository                                   |
| `stargazers_count` | counter, `uint64`          | count         | number of stars                                          |
| `forks_count`      | counter, `uint64`          | count         | number of forks                                          |
| `open_issues_count`| counter, `uint64`          | count         | number of open issues                                    |
| `languages`        | histogram, `uint64`        | byte          | languages of the source-code                             |
