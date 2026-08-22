package main

import (
	"time"
)

func (c *checker) securitySteps(history bool) []checkStep {
	gitleaksName := "Gitleaks current tree"
	gitleaksArgs := []string{"detect", "--no-git", "--no-banner", "--redact=100", "--source", "."}
	if history {
		gitleaksName = "Gitleaks reachable history"
		gitleaksArgs = []string{"detect", "--no-banner", "--redact=100", "--log-opts=--all", "--source", "."}
	}
	return []checkStep{
		{"gosec", c.runGosec},
		{gitleaksName, func() error { return c.runGitleaks(gitleaksName, gitleaksArgs...) }},
	}
}

func (c *checker) runSecurity() error {
	return c.runSequential(c.securitySteps(false))
}

func (c *checker) runSecurityHistory() error {
	return c.runSequential(c.securitySteps(true))
}

func (c *checker) runGosec() error {
	tool, err := c.tool("gosec")
	if err != nil {
		return err
	}
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	return c.runAnalysis("gosec", c.toolExecutable(tool), "-include=G101,G107,G108,G109,G110,G111,G112,G113,G114,G201,G202,G203,G401,G402,G403,G404,G501,G502,G503,G504,G505,G601,G602", "-nosec-require-rules", "-nosec-require-justification", "./...")
}

func (c *checker) runGitleaks(step string, arguments ...string) error {
	tool, err := c.tool("gitleaks")
	if err != nil {
		return err
	}
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	return c.runAnalysis(step, c.toolExecutable(tool), arguments...)
}

func (c *checker) runVulnerability() error {
	tool, err := c.tool("govulncheck")
	if err != nil {
		return err
	}
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	return c.runAnalysisWithEnvironment("govulncheck", c.vulnerabilityEnvironment(), c.toolExecutable(tool), "./...")
}

func (c *checker) vulnerabilityEnvironment() []string {
	return overrideEnvironment(c.offlineEnvironment(), map[string]string{
		"GOPROXY": "https://proxy.golang.org",
		"GOSUMDB": "sum.golang.org",
	})
}

func (c *checker) fuzzSteps() []checkStep {
	return []checkStep{
		{"fuzz GitHub Release reference", func() error {
			return c.runGoWithTimeout("fuzz GitHub Release reference", time.Minute, "test", "./internal/artifact/githubrelease", "-run=^$", "-fuzz=FuzzParseReference", "-fuzztime=5s")
		}},
		{"fuzz PyPI reference", func() error {
			return c.runGoWithTimeout("fuzz PyPI reference", time.Minute, "test", "./internal/artifact/pypi", "-run=^$", "-fuzz=FuzzParseReference", "-fuzztime=5s")
		}},
		{"fuzz wheel inspection", func() error {
			return c.runGoWithTimeout("fuzz wheel inspection", time.Minute, "test", "./internal/artifact/pypi", "-run=^$", "-fuzz=FuzzInspectWheelNoPanic", "-fuzztime=5s")
		}},
		{"fuzz sdist inspection", func() error {
			return c.runGoWithTimeout("fuzz sdist inspection", time.Minute, "test", "./internal/artifact/pypi", "-run=^$", "-fuzz=FuzzInspectSdistNoPanic", "-fuzztime=5s")
		}},
	}
}

func (c *checker) runFuzz() error {
	return c.runSequential(c.fuzzSteps())
}
