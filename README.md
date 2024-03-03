# git-cc: Conventional Commits Git Command

git-cc is interactive git sub-command that will help you craft beautify and informative commit message that adhere to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) standard.

---

<p style="text-align: center;"><b>
<a href="#installation">Installation</a>&nbsp&nbsp|&nbsp&nbsp<a href="#usage">Usage</a>&nbsp&nbsp|&nbsp&nbsp<a href="#configuration">Configuration</a></b>
</p> 
 
---  

## Installation

```sh
go install github.com/45413/git-cc
```

Ensure go bin directory to your path ``export PATH=${PATH}:$(go env GOPATH)/bin``

## Usage

To invoke simply run `git cc`

## Configuration

`git-cc` supports a simple yaml based configuration to customize the prompt behavoir on a repo basis. Simply add a `.git-cc.yaml` into the root of your repository. See [.git-cc.example.yaml](.git-cc.example.yaml)

```yaml
# .git-cc.yaml
use_defaults: true
custom_commit_types: [
  build,
  chore,
  ci,
  docs,
  style,
  refactor,
  perf,
  test
]
```

|      property       |                                           options                                           |
| :-----------------: | :-----------------------------------------------------------------------------------------: |
|    use_defaults     |                      If true use default commit types (default: true)                       |
| custom_commit_types | Custom commit types to include when prompting, appended to defaults if `use_defaults: true` |
