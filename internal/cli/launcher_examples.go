package cli

// Examples are inert, reviewed command syntax. They are never passed to Execute,
// a shell or a process. Help and validation come from the registered Cobra tree.
func launcherExamples(path string) []string {
	switch path {
	case "helox":
		return []string{"helox --help", "helox config language auto", "helox doctor"}
	case "helox npm inspect":
		return []string{"helox npm inspect is-number@7.0.0"}
	case "helox npm install":
		return []string{"helox npm install is-number@7.0.0 --target /absolute/path/new-project"}
	case "helox npm update":
		return []string{"helox npm update --target /absolute/path/managed-project"}
	case "helox pypi inspect":
		return []string{"helox pypi inspect packaging@25.0"}
	case "helox pip install":
		return []string{
			"helox pip install packaging@25.0",
			"helox pip install --upgrade packaging",
			"helox pip install torch@2.14.1 --source pytorch:cpu",
		}
	case "helox go get":
		return []string{"helox go get github.com/google/uuid@v1.6.0", "helox go get -u github.com/google/uuid"}
	case "helox go mod download":
		return []string{"helox go mod download"}
	case "helox go build":
		return []string{"helox go build ./..."}
	case "helox cargo add":
		return []string{"helox cargo add itoa@1.0.15"}
	case "helox cargo update":
		return []string{"helox cargo update", "helox cargo update -p itoa"}
	case "helox cargo build":
		return []string{"helox cargo build"}
	case "helox terraform init":
		return []string{"helox terraform init hashicorp/random@3.7.2", "helox terraform init -upgrade"}
	case "helox github inspect":
		return []string{"helox github inspect cli/cli@v2.102.0#gh_2.102.0_linux_amd64.tar.gz"}
	case "helox github install":
		return []string{
			"helox github install cli/cli@v2.102.0#gh_2.102.0_linux_amd64.tar.gz --target /absolute/path/new-gh",
			"helox github install cli/cli@v2.102.0#gh_2.102.0_linux_amd64.tar.gz --target /absolute/path/managed-gh --replace",
		}
	case "helox config language":
		return []string{"helox config language auto", "helox config language ko", "helox --lang en --help"}
	case "helox doctor":
		return []string{"helox doctor"}
	}
	return nil
}
