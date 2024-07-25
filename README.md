# coding-update-values-file

A simple tool designed to create a commit to update a key-value `JSON` file in a `coding.net` repository

## Usage

**Environment Variables**

- `CODING_USERNAME` - The username of project token
- `CODING_PASSWORD` - The password of project token

**Parameters**

- `--repo` - The repository name, e.g. `my-team/my-project/my-repo`
- `--branch` - The branch name, e.g. `master`
- `--file` - The file path, e.g. `build.json`
- `--key` - The key name, e.g. `version`
- `--value` - The value to update, e.g. `1.0.0`

## Credits

GUO YANKE, MIT License
