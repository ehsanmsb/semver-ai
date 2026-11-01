package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai/openai"
	"github.com/ehsanmsb/semver-ai/internal/analyzer"
	"github.com/ehsanmsb/semver-ai/internal/application"
	"github.com/ehsanmsb/semver-ai/internal/config"
	gitrepo "github.com/ehsanmsb/semver-ai/internal/git"
	"github.com/ehsanmsb/semver-ai/internal/gitlab"
	"github.com/ehsanmsb/semver-ai/internal/release"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var buildVersion = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: release-manager <analyze-mr|predict|prepare-release|publish-release|version|validate-config> [flags]")
		return 2
	}
	command := args[0]
	if command == "--help" || command == "help" {
		fmt.Fprintln(out, "Commands: analyze-mr, predict, prepare-release, publish-release, version, validate-config\nFlags follow the command; use COMMAND --help.")
		return 0
	}
	switch command {
	case "analyze-mr", "predict", "prepare-release", "publish-release", "version", "validate-config":
	default:
		fmt.Fprintln(errOut, "unknown command")
		return 2
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(errOut)
	path := flags.String("config", ".release-manager.yml", "configuration path")
	output := flags.String("output", "", "candidate path for prepare/publish; report path for other commands")
	asJSON := flags.Bool("json", false, "JSON output")
	verbose := flags.Bool("verbose", false, "structured diagnostic logging")
	dry := flags.Bool("dry-run", false, "perform validation/analysis without writing artifacts, comments, tags or releases")
	build := flags.Bool("build", false, "version: print tool build version only")
	if e := flags.Parse(args[1:]); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errOut, "unexpected positional arguments")
		return 2
	}
	level := slog.LevelWarn
	if *verbose {
		level = slog.LevelInfo
	}
	log := slog.New(slog.NewJSONHandler(errOut, &slog.HandlerOptions{Level: level}))
	failure := func(e error) int { log.Error("command failed", "error", e.Error()); return 1 }
	emit := func(v any, text string) error {
		var b []byte
		var e error
		if *asJSON {
			b, e = json.MarshalIndent(v, "", "  ")
			b = append(b, '\n')
		} else {
			b = []byte(text + "\n")
		}
		if e != nil {
			return e
		}
		if *output != "" && command != "prepare-release" && command != "publish-release" && !*dry {
			return release.Write(*output, b)
		}
		_, e = out.Write(b)
		return e
	}
	if command == "version" && *build {
		if e := emit(map[string]string{"build_version": buildVersion}, buildVersion); e != nil {
			return failure(e)
		}
		return 0
	}
	cfg, e := config.Load(*path)
	if e != nil {
		return failure(e)
	}
	if command == "validate-config" {
		if e = emit(map[string]string{"status": "VALID"}, "Configuration is valid"); e != nil {
			return failure(e)
		}
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	repo := gitrepo.Repository{Dir: "."}
	if command == "version" {
		head, e := repo.Head(ctx)
		if e != nil {
			return failure(e)
		}
		tag, e := repo.Latest(ctx, cfg.Release.TagPrefix, head)
		if e != nil {
			return failure(e)
		}
		if e = emit(map[string]string{"build_version": buildVersion, "current_tag": tag, "target_commit": head}, fmt.Sprintf("Build: %s\nCurrent tag: %s\nTarget: %s", buildVersion, tag, head)); e != nil {
			return failure(e)
		}
		return 0
	}
	key := os.Getenv(cfg.AI.KeyEnv)
	token := os.Getenv(cfg.GitLab.TokenEnv)
	secrets := []string{key, token}
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(name)
		if strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "API_KEY") {
			if value != "" {
				secrets = append(secrets, value)
			}
		}
	}
	svc := application.Service{Config: cfg, Repo: repo, Analyzer: analyzer.Analyzer{Config: cfg.Analysis, Secrets: secrets, Provider: openai.Provider{BaseURL: cfg.AI.BaseURL, Model: cfg.AI.Model, Key: key, Timeout: cfg.AI.Timeout, MaxTokens: cfg.Analysis.Limits.ContextTokens}}}
	log.Info("starting command", "command", command, "dry_run", *dry)
	var client *gitlab.Client
	if command != "predict" {
		project := os.Getenv("CI_PROJECT_ID")
		if project == "" {
			project = os.Getenv("SEMVER_PROJECT")
		}
		client, e = gitlab.New(cfg.GitLab.APIURL, token, project)
		if e != nil {
			return failure(e)
		}
		svc.Store = client
	}
	var report application.Report
	switch command {
	case "predict":
		report, e = svc.Predict(ctx)
	case "analyze-mr":
		iid, err := strconv.ParseInt(os.Getenv("CI_MERGE_REQUEST_IID"), 10, 64)
		if err != nil || iid <= 0 {
			return failure(fmt.Errorf("CI_MERGE_REQUEST_IID must be positive"))
		}
		mr, err := client.MergeRequest(ctx, iid)
		if err != nil {
			return failure(err)
		}
		if mr.TargetBranch != cfg.Release.Branch {
			return failure(fmt.Errorf("MR target must be configured release branch"))
		}
		snap, err := repo.Between(ctx, "", mr.Base, mr.Head)
		if err != nil {
			return failure(err)
		}
		// Git supplies the complete diff; API file sets must agree to detect omitted API pages.
		files := map[string]bool{}
		for _, f := range snap.Files {
			files[f] = true
		}
		for _, f := range mr.Files {
			if !files[f] {
				return failure(fmt.Errorf("MR API diff and local Git disagree"))
			}
		}
		snap.BaseTag, err = repo.Latest(ctx, cfg.Release.TagPrefix, mr.Base)
		if err != nil {
			return failure(err)
		}
		metadata := "MR title: " + mr.Title
		if cfg.Analysis.Context.MR {
			metadata += "\nMR description: " + mr.Description
		}
		report, e = svc.Analyze(ctx, snap, metadata)
		if e == nil && cfg.GitLab.Comment && !*dry {
			e = client.Comment(ctx, iid, fmt.Sprintf("Semver prediction (advisory): %s %s\n\n%s", report.Bump, report.Next, report.Notes))
		}
	case "prepare-release":
		candidatePath := *output
		if candidatePath == "" {
			candidatePath = "semver-release-candidate.json"
		}
		notesPath := filepath.Join(filepath.Dir(candidatePath), "RELEASE_NOTES.md")
		if !*dry {
			for _, p := range []string{candidatePath, notesPath} {
				if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
					return failure(err)
				}
			}
		}
		branch, err := repo.Branch(ctx)
		if err != nil {
			branch = os.Getenv("CI_COMMIT_BRANCH")
			head, he := repo.Head(ctx)
			if he != nil || os.Getenv("CI_COMMIT_SHA") != head {
				return failure(fmt.Errorf("detached checkout must match GitLab CI_COMMIT_SHA"))
			}
		}
		var candidate *release.Candidate
		report, candidate, e = svc.Prepare(ctx, branch)
		if e == nil && candidate != nil && !*dry {
			e = release.Write(notesPath, []byte(candidate.Notes))
			if e == nil {
				e = candidate.Save(candidatePath)
			}
		}
	case "publish-release":
		candidatePath := *output
		if candidatePath == "" {
			candidatePath = "semver-release-candidate.json"
		}
		candidate, err := release.Load(candidatePath)
		if err != nil {
			return failure(err)
		}
		authorized := os.Getenv("GITLAB_CI") == "true" && os.Getenv("CI_JOB_MANUAL") == "true" && os.Getenv("CI_COMMIT_REF_PROTECTED") == "true" && os.Getenv("CI_ENVIRONMENT_NAME") == "production-release"
		if os.Getenv("CI_PROJECT_ID") != strconv.FormatInt(candidate.ProjectID, 10) || os.Getenv("CI_COMMIT_SHA") != candidate.Target {
			return failure(fmt.Errorf("candidate differs from current CI project/commit"))
		}
		if head, he := repo.Head(ctx); he != nil || head != candidate.Target {
			return failure(fmt.Errorf("candidate target differs from checkout"))
		}
		e = (release.Publisher{Store: client, Config: cfg}).Publish(ctx, candidate, authorized, *dry)
		if e == nil {
			status := "PUBLISHED"
			if *dry {
				status = "VALIDATED"
			}
			e = emit(map[string]string{"status": status, "version": candidate.Next}, status+" "+candidate.Next)
		}
		if e != nil {
			return failure(e)
		}
		return 0
	}
	if e != nil {
		return failure(e)
	}
	if e = emit(report, fmt.Sprintf("%s: %s %s\n%s", report.Status, report.Bump, report.Next, report.Notes)); e != nil {
		return failure(e)
	}
	return 0
}
