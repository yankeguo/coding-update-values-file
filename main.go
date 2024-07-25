package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/yankeguo/rg"
)

type HybridCodingResponse struct {
	Response struct {
		Error struct {
			Message string `json:"Message"`
			Code    string `json:"Code"`
		}
		GitFile struct {
			Encoding string `json:"Encoding,omitempty"`
			Content  string `json:"Content,omitempty"`
		} `json:"GitFile"`
		Commits []struct {
			Sha string `json:"Sha"`
		} `json:"Commits"`
		GitCommit struct {
			Sha string `json:"Sha"`
		} `json:"GitCommit"`
	} `json:"Response"`
}

func invokeCodingAPI(ctx context.Context, client *resty.Client, action string, body map[string]any) (res HybridCodingResponse, err error) {
	defer rg.Guard(&err)

	resp := rg.Must(client.R().
		SetContext(ctx).
		SetBody(body).SetResult(&res).
		SetQueryParam("Action", action).
		Post(action),
	)

	if resp.IsError() {
		err = errors.New("coding api error: " + resp.String())
		return
	}

	if res.Response.Error.Code != "" || res.Response.Error.Message != "" {
		err = errors.New("coding api error: " + res.Response.Error.Code + ": " + res.Response.Error.Message)
		return
	}

	return
}

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Println("exited with error:", err.Error())
		os.Exit(1)
	}()
	defer rg.Guard(&err)

	var (
		optRepo   string
		optFile   string
		optKey    string
		optValue  string
		optBranch string
	)

	flag.StringVar(&optRepo, "repo", "", "coding repository, in the format of 'tenant/user/project'")
	flag.StringVar(&optFile, "file", "", "file to update, in the format of 'path/to/file'")
	flag.StringVar(&optBranch, "branch", "master", "branch")
	flag.StringVar(&optKey, "key", "", "key")
	flag.StringVar(&optValue, "value", "", "value")
	flag.Parse()

	ctx := context.Background()

	debug, _ := strconv.ParseBool(os.Getenv("CODING_DEBUG"))

	client := resty.New().
		SetBasicAuth(os.Getenv("CODING_USERNAME"), os.Getenv("CODING_PASSWORD")).
		SetBaseURL("https://e.coding.net/open-api").
		SetDebug(debug)

	resFile := rg.Must(invokeCodingAPI(ctx, client, "DescribeGitFile", map[string]any{
		"DepotPath": optRepo,
		"Path":      optFile,
		"Ref":       optBranch,
	}))

	if resFile.Response.GitFile.Encoding != "base64" {
		err = errors.New("file not found or unsupported encoding")
		return
	}

	content := rg.Must(base64.StdEncoding.DecodeString(resFile.Response.GitFile.Content))

	var values map[string]any

	rg.Must0(json.Unmarshal(content, &values))

	if values[optKey] == optValue {
		log.Println("value not changed")
		return
	}

	values[optKey] = optValue

	content = rg.Must(json.MarshalIndent(values, "", "  "))

	resCommits := rg.Must(invokeCodingAPI(ctx, client, "DescribeGitCommitInfos", map[string]any{
		"DepotPath":  optRepo,
		"Ref":        optBranch,
		"PageNumber": 1,
		"PageSize":   1,
	}))

	if len(resCommits.Response.Commits) == 0 {
		err = errors.New("no commit found")
		return
	}

	lastSha := resCommits.Response.Commits[0].Sha

	resModify := rg.Must(invokeCodingAPI(ctx, client, "ModifyGitFiles", map[string]any{
		"DepotPath":     optRepo,
		"Ref":           optBranch,
		"LastCommitSha": lastSha,
		"Message":       "chore: update " + optFile,
		"GitFiles": []map[string]any{{
			"Path":    optFile,
			"Content": string(content),
		}},
	}))

	if resModify.Response.GitCommit.Sha == "" {
		err = errors.New("failed modifying git file")
		return
	}

	log.Println("updated file", optFile)
}
