package config

import (
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Version      int      `yaml:"version" json:"version"`
	Release      Release  `yaml:"release" json:"release"`
	Analysis     Analysis `yaml:"analysis" json:"analysis"`
	AI           AI       `yaml:"ai" json:"ai"`
	GitLab       GitLab   `yaml:"gitlab" json:"gitlab"`
	ReleaseNotes struct {
		Sections []string `yaml:"sections" json:"sections"`
	} `yaml:"release_notes" json:"release_notes"`
}
type Release struct {
	Branch         string `yaml:"branch" json:"branch"`
	TagPrefix      string `yaml:"tag_prefix" json:"tag_prefix"`
	InitialVersion string `yaml:"initial_version" json:"initial_version"`
	Pre1           struct {
		Breaking string `yaml:"breaking_change_bump" json:"breaking_change_bump"`
	} `yaml:"pre_1_0" json:"pre_1_0"`
	Approval struct {
		Required bool `yaml:"required" json:"required"`
	} `yaml:"approval" json:"approval"`
	Rules struct {
		Performance string `yaml:"performance" json:"performance"`
		Docs        string `yaml:"docs" json:"docs"`
		Refactor    string `yaml:"refactor" json:"refactor"`
		Chore       string `yaml:"chore" json:"chore"`
		Tests       string `yaml:"tests" json:"tests"`
	} `yaml:"rules" json:"rules"`
}
type Analysis struct {
	Confidence float64 `yaml:"confidence_threshold" json:"confidence_threshold"`
	Context    struct {
		Commits bool `yaml:"include_commit_messages" json:"include_commit_messages"`
		MR      bool `yaml:"include_mr_description" json:"include_mr_description"`
		Files   bool `yaml:"include_repository_files" json:"include_repository_files"`
	} `yaml:"context" json:"context"`
	Limits struct {
		DiffBytes     int `yaml:"max_diff_bytes" json:"max_diff_bytes"`
		ContextTokens int `yaml:"max_context_tokens" json:"max_context_tokens"`
	} `yaml:"limits" json:"limits"`
}
type AI struct {
	MaxCompletionTokens int           `yaml:"max_completion_tokens" json:"max_completion_tokens"`
	Provider            string        `yaml:"provider" json:"provider"`
	Model               string        `yaml:"model" json:"model"`
	BaseURL             string        `yaml:"base_url" json:"base_url"`
	KeyEnv              string        `yaml:"api_key_env" json:"api_key_env"`
	Timeout             time.Duration `yaml:"timeout" json:"timeout"`
}
type GitLab struct {
	APIURL        string `yaml:"api_url" json:"api_url"`
	TokenEnv      string `yaml:"token_env" json:"token_env"`
	Comment       bool   `yaml:"create_mr_comment" json:"create_mr_comment"`
	CreateRelease bool   `yaml:"create_release" json:"create_release"`
}

func Defaults() Config {
	var c Config
	c.Version = 1
	c.Release.Branch = "main"
	c.Release.TagPrefix = "v"
	c.Release.InitialVersion = "0.1.0"
	c.Release.Pre1.Breaking = "minor"
	c.Release.Approval.Required = true
	c.Release.Rules.Performance = "patch"
	c.Release.Rules.Docs = "none"
	c.Release.Rules.Refactor = "none"
	c.Release.Rules.Chore = "none"
	c.Release.Rules.Tests = "none"
	c.Analysis.Confidence = .8
	c.Analysis.Context.Commits = true
	c.Analysis.Context.MR = true
	c.Analysis.Context.Files = true
	c.Analysis.Limits.DiffBytes = 500000
	c.Analysis.Limits.ContextTokens = 30000
	c.AI = AI{MaxCompletionTokens: 4096, Provider: "openai-compatible", BaseURL: "https://api.openai.com/v1", KeyEnv: "SEMVER_AI_API_KEY", Timeout: 120 * time.Second}
	c.GitLab = GitLab{APIURL: "https://gitlab.com/api/v4", TokenEnv: "GITLAB_TOKEN", Comment: false, CreateRelease: true}
	c.ReleaseNotes.Sections = []string{"breaking_changes", "features", "bug_fixes", "performance", "other"}
	return c
}
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return Decode(f)
}
func Decode(r io.Reader) (Config, error) {
	c := Defaults()
	data, err := io.ReadAll(io.LimitReader(r, 262145))
	if err != nil {
		return c, err
	}
	if len(data) > 262144 {
		return c, fmt.Errorf("config exceeds 256 KiB")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return c, fmt.Errorf("config is empty")
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return c, fmt.Errorf("invalid configuration YAML")
	}
	var check func(*yaml.Node) error
	check = func(n *yaml.Node) error {
		if n.Tag == "!!null" || n.Kind == yaml.AliasNode {
			return fmt.Errorf("null values and YAML aliases are not supported")
		}
		for _, child := range n.Content {
			if err := check(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(&node); err != nil {
		return c, err
	}
	d := yaml.NewDecoder(strings.NewReader(string(data)))
	d.KnownFields(true)
	if err = d.Decode(&c); err != nil {
		return c, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("config must contain exactly one YAML document")
	}
	return c, c.Validate()
}

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")) && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
func StableVersion(s string) bool {
	v, e := semver.StrictNewVersion(s)
	return e == nil && v.Prerelease() == "" && v.Metadata() == ""
}
func ValidRef(s string) bool {
	return s != "" && !strings.HasPrefix(s, "-") && !strings.HasPrefix(s, "/") && !strings.HasSuffix(s, "/") && !strings.ContainsAny(s, " ~^:?*[\\\x00\r\n\t") && !strings.Contains(s, "..") && !strings.Contains(s, "@{") && !strings.Contains(s, "//") && !strings.HasSuffix(s, ".") && !strings.Contains(s, ".lock") && !strings.Contains(s, "/.") && !strings.HasPrefix(s, ".") && s != "@"
}
func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported configuration version")
	}
	if !ValidRef(c.Release.Branch) || !ValidRef(c.Release.TagPrefix+"1.0.0") {
		return fmt.Errorf("invalid release branch or tag prefix")
	}
	if !StableVersion(c.Release.InitialVersion) {
		return fmt.Errorf("initial_version must be a stable strict SemVer")
	}
	if c.Release.Pre1.Breaking != "minor" && c.Release.Pre1.Breaking != "major" {
		return fmt.Errorf("pre_1_0 breaking_change_bump must be minor or major")
	}
	if !c.Release.Approval.Required {
		return fmt.Errorf("human approval cannot be disabled in the MVP")
	}
	for _, rule := range []string{c.Release.Rules.Performance, c.Release.Rules.Docs, c.Release.Rules.Refactor, c.Release.Rules.Chore, c.Release.Rules.Tests} {
		if rule != "none" && rule != "patch" && rule != "minor" && rule != "major" {
			return fmt.Errorf("invalid classification bump policy")
		}
	}
	if math.IsNaN(c.Analysis.Confidence) || math.IsInf(c.Analysis.Confidence, 0) || c.Analysis.Confidence <= 0 || c.Analysis.Confidence > 1 {
		return fmt.Errorf("confidence_threshold must be > 0 and <= 1")
	}
	if c.Analysis.Limits.DiffBytes < 1024 || c.Analysis.Limits.DiffBytes > 10000000 || c.Analysis.Limits.ContextTokens < 8192 || c.Analysis.Limits.ContextTokens > 1000000 {
		return fmt.Errorf("limits: diff bytes 1024..10000000; context tokens 8192..1000000")
	}
	if c.AI.Provider != "openai-compatible" || strings.TrimSpace(c.AI.Model) == "" {
		return fmt.Errorf("set an AI model and provider openai-compatible")
	}
	if !ValidURL(c.AI.BaseURL) || !ValidURL(c.GitLab.APIURL) {
		return fmt.Errorf("API URLs must be HTTPS (HTTP allowed only for loopback), without credentials, query or fragment")
	}
	if !envName.MatchString(c.AI.KeyEnv) || !envName.MatchString(c.GitLab.TokenEnv) {
		return fmt.Errorf("invalid credential environment variable name")
	}
	if c.AI.MaxCompletionTokens < 1 || c.AI.MaxCompletionTokens > 131072 {
		return fmt.Errorf("ai.max_completion_tokens must be 1..131072 (also subject to provider/model limits)")
	}
	if c.AI.Timeout < time.Second || c.AI.Timeout > 10*time.Minute {
		return fmt.Errorf("AI timeout must be 1s..10m")
	}
	if !c.GitLab.CreateRelease {
		return fmt.Errorf("create_release must be true in the MVP")
	}
	seen := map[string]bool{}
	for _, s := range c.ReleaseNotes.Sections {
		if seen[s] {
			return fmt.Errorf("duplicate release notes section")
		}
		seen[s] = true
	}
	for _, s := range []string{"breaking_changes", "features", "bug_fixes", "performance", "other"} {
		if !seen[s] {
			return fmt.Errorf("release notes must include all five sections (order is configurable)")
		}
	}
	if len(seen) != 5 {
		return fmt.Errorf("unknown release notes section")
	}
	return nil
}
