package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	projectbuild "github.com/rahoney/heliopause/internal/artifact/projectbuild"
	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestLinuxCargoDependencyFreeBuildCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_CARGO_BUILD_CLI_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Cargo qualification requires Linux amd64")
	}
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_cargo_build_fixture'\nversion='0.1.0'\nedition='2021'\n")
	lock := []byte("version = 4\n[[package]]\nname = 'haa_cargo_build_fixture'\nversion = '0.1.0'\n")
	for name, body := range map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock, "src/main.rs": []byte("fn main() { println!(\"qualified\"); }\n")} {
		if err := os.WriteFile(filepath.Join(project, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var stdout, stderr bytes.Buffer
	err := bootstrap.Run(ctx, []string{"cargo", "build"}, &stdout, &stderr)
	for name, expected := range map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock} {
		actual, readErr := os.ReadFile(filepath.Join(project, name))
		if readErr != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("build changed %s: %v", name, readErr)
		}
	}
	if _, lockErr := os.Lstat(filepath.Join(project, ".heliopause-cargo-transaction.lock")); !os.IsNotExist(lockErr) {
		t.Fatal("build retained project guard")
	}
	if err != nil {
		entries, e := os.ReadDir(filepath.Join(project, ".heliopause", "builds"))
		if !os.IsNotExist(e) && (e != nil || len(entries) != 0) {
			t.Fatal("failed build published output")
		}
		t.Fatalf("actual Cargo build failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Policy: ALLOW") || !strings.Contains(stdout.String(), "Build output: .heliopause/builds/") {
		t.Fatalf("build lacks approved published output: %s", stdout.String())
	}
	entries, err := os.ReadDir(filepath.Join(project, ".heliopause", "builds"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("build output count: %v", err)
	}
	output := filepath.Join(project, ".heliopause", "builds", entries[0].Name())
	program, err := os.ReadFile(filepath.Join(output, "haa_cargo_build_fixture"))
	if err != nil || len(program) < 4 || !bytes.Equal(program[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("exact compiler output is unavailable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, ".haa-build.json")); err != nil {
		t.Fatal("build lacks independent Evidence binding receipt")
	}
}

func cargoBuildFixtureCLI(t *testing.T, files map[string]string, outputs map[string]string) {
	cargoBuildFixtureCLIExpect(t, files, outputs, "")
}

func cargoBuildFixtureCLIExpect(t *testing.T, files map[string]string, outputs map[string]string, failure string) {
	cargoBuildSelectedFixtureCLI(t, files, outputs, failure, "")
}

func cargoBuildSelectedFixtureCLI(t *testing.T, files map[string]string, outputs map[string]string, failure, reference string) {
	t.Helper()
	if os.Getenv("HELOX_CARGO_BUILD_CLI_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Cargo qualification requires Linux amd64")
	}
	root := t.TempDir()
	project := filepath.Join(root, "project")
	for name, body := range files {
		destination := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if reference != "" {
		var stdout, stderr bytes.Buffer
		if err := bootstrap.Run(ctx, []string{"cargo", "add", reference}, &stdout, &stderr); err != nil {
			t.Fatalf("public dependency selection failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		}
		for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
			body, err := os.ReadFile(filepath.Join(project, name))
			if err != nil {
				t.Fatal(err)
			}
			files[name] = string(body)
		}
	}
	// Host settings must not enter the controller's fixed isolated environment.
	t.Setenv("CARGO_HOME", filepath.Join(root, "ambient-poison-home"))
	t.Setenv("RUSTC", "/haa-forbidden-ambient-rustc")
	t.Setenv("RUSTC_WRAPPER", "/haa-forbidden-ambient-wrapper")
	t.Setenv("RUSTFLAGS", "--cfg haa_forbidden_ambient")
	t.Setenv("CARGO_TARGET_DIR", filepath.Join(root, "ambient-poison-target"))
	repeats := 2
	if failure != "" {
		repeats = 1
	}
	seen := map[string]bool{}
	previousReceipts := map[string][]byte{}
	for attempt := range repeats {
		var stdout, stderr bytes.Buffer
		err := bootstrap.Run(ctx, []string{"cargo", "build"}, &stdout, &stderr)
		for name, body := range files {
			actual, readErr := os.ReadFile(filepath.Join(project, name))
			if readErr != nil || string(actual) != body {
				t.Fatalf("build changed source/control %s: %v", name, readErr)
			}
		}
		if _, lockErr := os.Lstat(filepath.Join(project, ".heliopause-cargo-transaction.lock")); !os.IsNotExist(lockErr) {
			t.Fatal("build retained guard")
		}
		if err != nil {
			entries, e := os.ReadDir(filepath.Join(project, ".heliopause", "builds"))
			if !os.IsNotExist(e) && (e != nil || len(entries) != 0) {
				t.Fatal("failed build published output")
			}
			if failure != "" && strings.Contains(err.Error()+stdout.String(), failure) && !strings.Contains(stdout.String(), "Policy: ALLOW") && !strings.Contains(stdout.String(), "Build output:") {
				t.Logf("actual Cargo build denied: %v stdout=%s", err, stdout.String())
				return
			}
			t.Fatalf("actual Cargo build failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		}
		if failure != "" {
			t.Fatalf("unsafe Cargo build completed: stdout=%s", stdout.String())
		}
		if !strings.Contains(stdout.String(), "Policy: ALLOW") || !strings.Contains(stdout.String(), "Build output: .heliopause/builds/") {
			t.Fatal("build lacks actual ALLOW/publication")
		}
		entries, err := os.ReadDir(filepath.Join(project, ".heliopause", "builds"))
		if err != nil || len(entries) != attempt+1 {
			t.Fatal("build output count differs")
		}
		directory := ""
		for _, line := range strings.Split(stdout.String(), "\n") {
			if suffix, ok := strings.CutPrefix(line, "Build output: "); ok {
				directory = suffix
			}
		}
		if !strings.HasPrefix(directory, ".heliopause/builds/run_") || seen[directory] {
			t.Fatalf("output destination missing/reused: %q", directory)
		}
		seen[directory] = true
		output := filepath.Join(project, directory)
		for name, prefix := range outputs {
			body, err := os.ReadFile(filepath.Join(output, name))
			if err != nil || !bytes.HasPrefix(body, []byte(prefix)) {
				t.Fatalf("compiler output %s unavailable: %v", name, err)
			}
		}
		verifyCargoBuildReceipt(t, root, project, directory, len(outputs))
		for previous, expected := range previousReceipts {
			actual, err := os.ReadFile(filepath.Join(project, previous, ".haa-build.json"))
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatal("retained build replaced prior receipt")
			}
			verifyCargoBuildReceipt(t, root, project, previous, len(outputs))
		}
		receipt, err := os.ReadFile(filepath.Join(output, ".haa-build.json"))
		if err != nil {
			t.Fatal(err)
		}
		previousReceipts[directory] = receipt
	}
}

func verifyCargoBuildReceipt(t *testing.T, root, project, directory string, outputCount int) {
	t.Helper()
	var receipt struct {
		Schema        int                                   `json:"schema"`
		Run           string                                `json:"run"`
		Source        string                                `json:"source_sha256"`
		Cache         string                                `json:"cache_sha256"`
		Graph         string                                `json:"graph_sha256"`
		Output        string                                `json:"output_sha256"`
		Recipe        string                                `json:"recipe_sha256"`
		Policy        string                                `json:"policy"`
		PolicyVersion uint64                                `json:"policy_version"`
		Files         []projectbuild.BuildOutputFile        `json:"files"`
		Evidence      []struct{ ID, Handle, SHA256 string } `json:"evidence"`
	}
	body, err := os.ReadFile(filepath.Join(project, directory, ".haa-build.json"))
	if err != nil || json.Unmarshal(body, &receipt) != nil {
		t.Fatal("build receipt invalid")
	}
	if receipt.Schema != 1 || receipt.Policy == "" || receipt.PolicyVersion == 0 || receipt.Run != filepath.Base(directory) || len(receipt.Files) != outputCount || len(receipt.Evidence) != 3 {
		t.Fatal("Run/Policy/output/Evidence coverage differs")
	}
	for _, digest := range []string{receipt.Source, receipt.Cache, receipt.Graph, receipt.Output, receipt.Recipe} {
		decoded, err := hex.DecodeString(digest)
		if err != nil || len(decoded) != 32 {
			t.Fatal("build digest binding missing")
		}
	}
	for _, file := range receipt.Files {
		body, err := os.ReadFile(filepath.Join(project, directory, file.Name))
		if err != nil || int64(len(body)) != file.Size {
			t.Fatal("published output size differs")
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatal("published output hash differs")
		}
	}
	seenEvidence := map[string]bool{}
	for _, record := range receipt.Evidence {
		body, err := os.ReadFile(filepath.Join(root, "cache", "heliopause", "evidence", receipt.Run, record.ID+".json"))
		if err != nil || seenEvidence[record.ID] || record.Handle != "evidence:"+receipt.Run+":"+record.ID {
			t.Fatal("independent Evidence unavailable/substituted")
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != record.SHA256 {
			t.Fatal("independent Evidence hash differs")
		}
		seenEvidence[record.ID] = true
	}
	t.Logf("actual Cargo build run=%s source=%s cache=%s graph=%s output=%s recipe=%s", receipt.Run, receipt.Source, receipt.Cache, receipt.Graph, receipt.Output, receipt.Recipe)
}

func TestLinuxCargoPublicBuildCLIIntegration(t *testing.T) {
	cargoBuildSelectedFixtureCLI(t, map[string]string{
		"Cargo.toml":  "[package]\nname='haa_cargo_public'\nversion='0.1.0'\nedition='2021'\n",
		"src/main.rs": "fn main() { let mut b = itoa::Buffer::new(); println!(\"{}\", b.format(42)); }\n",
	}, map[string]string{"haa_cargo_public": "\x7fELF"}, "", "itoa@1.0.17")
}

func TestLinuxCargoTransitiveBuildCLIIntegration(t *testing.T) {
	cargoBuildSelectedFixtureCLI(t, map[string]string{
		"Cargo.toml":  "[package]\nname='haa_cargo_transitive'\nversion='0.1.0'\nedition='2021'\n",
		"src/main.rs": "fn main() { let _: serde::de::value::Error = serde::de::Error::custom(\"qualified\"); }\n",
	}, map[string]string{"haa_cargo_transitive": "\x7fELF"}, "", "serde@1.0.228")
}

func TestLinuxCargoLibraryBuildCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLI(t, map[string]string{
		"Cargo.toml": "[package]\nname='haa_cargo_library'\nversion='0.1.0'\nedition='2021'\n",
		"Cargo.lock": "version=4\n[[package]]\nname='haa_cargo_library'\nversion='0.1.0'\n",
		"src/lib.rs": "pub fn answer() -> u32 { 42 }\n",
	}, map[string]string{"libhaa_cargo_library.rlib": "!<arch>\n"})
}

func TestLinuxCargoWorkspaceBuildCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLI(t, map[string]string{
		"Cargo.toml":         "[workspace]\nmembers=['app','support']\nresolver='2'\n",
		"Cargo.lock":         "version=4\n[[package]]\nname='haa_workspace_app'\nversion='0.1.0'\ndependencies=['haa_workspace_support']\n[[package]]\nname='haa_workspace_support'\nversion='0.1.0'\n",
		"app/Cargo.toml":     "[package]\nname='haa_workspace_app'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nhaa_workspace_support={path='../support'}\n",
		"app/src/main.rs":    "fn main() { println!(\"{}\", haa_workspace_support::answer()); }\n",
		"support/Cargo.toml": "[package]\nname='haa_workspace_support'\nversion='0.1.0'\nedition='2021'\n",
		"support/src/lib.rs": "pub fn answer() -> u32 { 42 }\n",
	}, map[string]string{"haa_workspace_app": "\x7fELF", "libhaa_workspace_support.rlib": "!<arch>\n"})
}

func TestLinuxCargoBuildScriptBuildCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLI(t, map[string]string{
		"Cargo.toml":  "[package]\nname='haa_cargo_script'\nversion='0.1.0'\nedition='2021'\n",
		"Cargo.lock":  "version=4\n[[package]]\nname='haa_cargo_script'\nversion='0.1.0'\n",
		"build.rs":    "fn main() { println!(\"cargo:rustc-env=HAA_BUILD_VALUE=observed\"); }\n",
		"src/main.rs": "fn main() { println!(\"{}\", env!(\"HAA_BUILD_VALUE\")); }\n",
	}, map[string]string{"haa_cargo_script": "\x7fELF"})
}

func TestLinuxCargoProcMacroBuildCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLI(t, map[string]string{
		"Cargo.toml":        "[package]\nname='haa_cargo_macro_app'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nhaa_cargo_macros={path='macros'}\n",
		"Cargo.lock":        "version=4\n[[package]]\nname='haa_cargo_macro_app'\nversion='0.1.0'\ndependencies=['haa_cargo_macros']\n[[package]]\nname='haa_cargo_macros'\nversion='0.1.0'\n",
		"src/main.rs":       "haa_cargo_macros::make!(); fn main() { println!(\"{}\", made()); }\n",
		"macros/Cargo.toml": "[package]\nname='haa_cargo_macros'\nversion='0.1.0'\nedition='2021'\n[lib]\nproc-macro=true\n",
		"macros/src/lib.rs": "use proc_macro::TokenStream; #[proc_macro] pub fn make(_: TokenStream) -> TokenStream { \"fn made()->u32{42}\".parse().unwrap() }\n",
	}, map[string]string{"haa_cargo_macro_app": "\x7fELF"})
}

func cargoBuildScriptFixture(script string) map[string]string {
	return map[string]string{
		"Cargo.toml":  "[package]\nname='haa_cargo_security'\nversion='0.1.0'\nedition='2021'\n",
		"Cargo.lock":  "version=4\n[[package]]\nname='haa_cargo_security'\nversion='0.1.0'\n",
		"build.rs":    script,
		"src/main.rs": "fn main() {}\n",
	}
}

func TestLinuxCargoBuildFilesystemSecurityCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLIExpect(t, cargoBuildScriptFixture("fn main() { let _ = std::fs::read(\"/etc/passwd\"); }\n"), nil, "STREAM_FAULT")
}

func TestLinuxCargoBuildNetworkSecurityCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLIExpect(t, cargoBuildScriptFixture("fn main() { let _ = std::net::TcpStream::connect(\"192.0.2.1:443\"); }\n"), nil, "M3_NETWORK_ATTEMPT")
}

func TestLinuxCargoBuildConfigurationSecurityCLIIntegration(t *testing.T) {
	for name, script := range map[string]string{
		"cargo-home":        "use std::io::Write; fn main() { if let Ok(mut f) = std::fs::OpenOptions::new().append(true).open(format!(\"{}/config.toml\", std::env::var(\"CARGO_HOME\").unwrap())) { let _ = f.write_all(b\"\\n# untrusted mutation\\n\"); } }\n",
		"working-directory": "fn main() { std::fs::create_dir_all(\"/tmp/.cargo\").unwrap(); std::fs::write(\"/tmp/.cargo/config.toml\", b\"[net]\\noffline=true\\n\").unwrap(); }\n",
	} {
		t.Run(name, func(t *testing.T) {
			cargoBuildFixtureCLIExpect(t, cargoBuildScriptFixture(script), nil, "M3_FILESYSTEM_VIOLATION")
		})
	}
}

func TestLinuxCargoBuildUnexpectedChildSecurityCLIIntegration(t *testing.T) {
	cargoBuildFixtureCLIExpect(t, cargoBuildScriptFixture("fn main() { let _ = std::process::Command::new(\"/bin/sh\").arg(\"-c\").arg(\"true\").status(); }\n"), nil, "STREAM_FAULT")
}

func TestLinuxCargoInvalidSourceBuildCLIIntegration(t *testing.T) {
	files := cargoBuildScriptFixture("fn main() {}\n")
	files["src/main.rs"] = "fn main() { undefined_function(); }\n"
	cargoBuildFixtureCLIExpect(t, files, nil, "command_status=EXIT_STATUS_101")
}

func TestLinuxCargoProcMacroSecurityCLIIntegration(t *testing.T) {
	for name, action := range map[string]string{
		"filesystem": `let _ = std::fs::read("/etc/passwd");`,
		"network":    `let _ = std::net::TcpStream::connect("192.0.2.1:443");`,
	} {
		t.Run(name, func(t *testing.T) {
			failure := "STREAM_FAULT"
			if name == "network" {
				failure = "M3_NETWORK_ATTEMPT"
			}
			cargoBuildFixtureCLIExpect(t, map[string]string{
				"Cargo.toml":        "[package]\nname='haa_macro_security'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nhaa_macros={path='macros'}\n",
				"Cargo.lock":        "version=4\n[[package]]\nname='haa_macro_security'\nversion='0.1.0'\ndependencies=['haa_macros']\n[[package]]\nname='haa_macros'\nversion='0.1.0'\n",
				"src/main.rs":       "haa_macros::make!(); fn main() {}\n",
				"macros/Cargo.toml": "[package]\nname='haa_macros'\nversion='0.1.0'\nedition='2021'\n[lib]\nproc-macro=true\n",
				"macros/src/lib.rs": "use proc_macro::TokenStream; #[proc_macro] pub fn make(_: TokenStream) -> TokenStream { " + action + " TokenStream::new() }\n",
			}, nil, failure)
		})
	}
}
