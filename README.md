# GitHub Delete Artifacts

Delete artifacts uploaded by GitHub Actions to free up storage spending

## Usage

### Use prebuilt binary

Download from [releases](https://github.com/sarisia/github-delete-artifacts/releases) then

```
./github-delete-artifacts
```

### go run

Clone this repository and

```
go run .
```

## Environment Variables

* **DA_TOKEN** (*required*) - Valid GitHub access token with `repo` scope
* **DA_REPO**  (*required*) - Target repository like: `sarisia/github-delete-artifacts`
