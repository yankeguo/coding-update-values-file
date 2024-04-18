package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"github.com/go-resty/resty/v2"
	"github.com/yankeguo/rg"
	"log"
	"os"
)

type DescribeGitFileRequest struct {
	DepotPath string `json:"DepotPath"`
	Ref       string `json:"Ref"`
	Path      string `json:"Path"`
}

type GitFile struct {
	FileName      string `json:"FileName,omitempty"`
	FilePath      string `json:"FilePath,omitempty"`
	Size          int64  `json:"Size,omitempty"`
	Encoding      string `json:"Encoding,omitempty"`
	Content       string `json:"Content,omitempty"`
	ContentSha256 string `json:"ContentSha256,omitempty"`
	Sha           string `json:"Sha,omitempty"`
}

type DescribeGitFileResponse struct {
	GitFile GitFile `json:"GitFile"`
}

type ModifyGitFile struct {
	Path    string `json:"Path"`
	Content string `json:"Content"`
}

type ModifyGitFilesRequest struct {
	DepotPath     string          `json:"DepotPath,omitempty"`
	GitFiles      []ModifyGitFile `json:"GitFiles,omitempty"`
	LastCommitSha string          `json:"LastCommitSha,omitempty"`
	Message       string          `json:"Message,omitempty"`
	Ref           string          `json:"Ref,omitempty"`
}

type ModifyGitFilesResponse struct{}

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

	client := resty.New().
		SetBasicAuth(os.Getenv("CODING_USERNAME"), os.Getenv("CODING_PASSWORD")).
		SetBaseURL("https://e.coding.net/open-api")

	var file GitFile

	{
		var res DescribeGitFileResponse

		resp := rg.Must(client.R().
			SetContext(ctx).
			SetBody(&DescribeGitFileRequest{
				DepotPath: optRepo,
				Path:      optFile,
				Ref:       optBranch,
			}).SetResult(&res).
			SetQueryParam("Action", "DescribeGitFile").
			Post("DescribeGitFile"),
		)

		if resp.IsError() {
			err = errors.New("failed describing git file: " + resp.String())
			return
		}

		file = res.GitFile
	}

	log.Println("file fetched:", file.FilePath, file.Sha)

	if file.Encoding != "base64" {
		err = errors.New("unsupported encoding: " + file.Encoding)
		return
	}

	buf := rg.Must(base64.StdEncoding.DecodeString(file.Content))

	var m map[string]any

	rg.Must0(json.Unmarshal(buf, &m))

	if m[optKey] == optValue {
		log.Println("key already set to value")
		return
	}

	m[optKey] = optValue

	buf = rg.Must(json.MarshalIndent(m, "", "  "))

	{
		var res ModifyGitFilesResponse

		resp := rg.Must(client.R().
			SetContext(ctx).
			SetBody(&ModifyGitFilesRequest{
				DepotPath:     optRepo,
				Ref:           optBranch,
				LastCommitSha: file.Sha,
				Message:       "update " + file.FilePath,
				GitFiles: []ModifyGitFile{
					{
						Path:    file.FilePath,
						Content: string(buf),
					},
				},
			}).
			SetResult(&res).
			SetQueryParam("Action", "ModifyGitFiles").
			Post("ModifyGitFiles"),
		)

		if resp.IsError() {
			err = errors.New("failed modifying git file: " + resp.String())
			return
		}
	}
}
