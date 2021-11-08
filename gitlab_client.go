package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type GitlabClient struct {
	*gitlab.Client
	accessToken *AccessToken
}

func (client *GitlabClient) CheckMatch(project *gitlab.Project, filesToProcess []MatchFile, storage *Storage) {
	var (
		ok       = false
		records  []MatchRecord
		messages []Message
	)
	for _, fileToMatch := range filesToProcess {
		if record, reason, matched := getMatch(fileToMatch); matched {
			records = append(records, record)
			ok = true

			msg := Message{
				MatchType:       record.MatchType,
				ProjectID:       project.ID,
				ProjectName:     project.Namespace.Name + "/" + project.Name,
				MatchURL:        project.WebURL + "/-/blob/" + project.DefaultBranch + "/" + fileToMatch.Path,
				Path:            fileToMatch.Path,
				Filename:        record.MatchedFilename,
				RawMatchContent: record.MatchedContent,
				MatchName:       reason,
				Confidence:      record.Confidence,
				commitInfo:      fileToMatch.CommitInfo,
			}
			logrus.Printf("Got match in %s. Line: %d. Path: %s", fileToMatch.Filename, record.MatchedLineNumbers, fileToMatch.Path)
			messages = append(messages, msg)
		}
	}
	if ok {
		report, err := createReport(messages, storage)
		if err != nil {
			logrus.Println(err)
		}
		if ok, err := saveAndSendReport(report, project.Name, storage); !ok || err != nil {
			logrus.Println(err)
		}

	}
	lastActivityBytes := make([]byte, 8)
	lastActivity := uint64(project.LastActivityAt.UnixMilli())
	binary.BigEndian.PutUint64(lastActivityBytes, lastActivity)
	activityKey := make([]byte, 8)
	binary.BigEndian.PutUint64(activityKey, uint64(project.ID))

}

func (client *GitlabClient) getProject(projectID int) *gitlab.Project {
	var project *gitlab.Project
	var err error
	for {
		project, _, err = client.Projects.GetProject(projectID, &gitlab.GetProjectOptions{})
		if !isTimeout(err) {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return project
}

func (client *GitlabClient) isProjectActive(projectID int) bool {
	timeDelta := gitlab.ISOTime(time.Now().AddDate(0, -1, 0))

	pushed := gitlab.PushedEventType
	listContributionEventsOpts := &gitlab.ListContributionEventsOptions{
		After:  &timeDelta,
		Action: &pushed,
	}
	var projectEvents []*gitlab.ContributionEvent
	var err error
	for {
		projectEvents, _, err = client.Events.ListProjectVisibleEvents(projectID, listContributionEventsOpts)
		if !isTimeout(err) {
			break
		}
		time.Sleep(5 * time.Second)
	}

	return !(projectEvents == nil || len(projectEvents) == 0)
}

func (client *GitlabClient) processProjects(checker *Checker, projectsChan chan int) (wait <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		var projectGroup sync.WaitGroup

		for projectID := range projectsChan {
			if client.isProjectActive(projectID) {
				projectGroup.Add(1)
				go client.processProject(&projectGroup, projectID, checker)
			}
		}
		projectGroup.Wait()
		close(ch)
	}()
	return ch
}

func (client *GitlabClient) recursiveListTree(pid int, options *gitlab.ListTreeOptions) ([]*gitlab.TreeNode, *gitlab.Response, error) {
	var tree []*gitlab.TreeNode
	options.PerPage = 20
	options.Page = 1
	var err error
	for {
		var ctree []*gitlab.TreeNode
		var resp *gitlab.Response
		for {
			ctree, resp, err = client.Repositories.ListTree(pid, options)
			if !isTimeout(err) {
				break
			}
			time.Sleep(5 * time.Second)
		}
		if err != nil {
			logrus.Printf("list-tree error %s", err)
			return tree, resp, err
		}
		for _, node := range ctree {
			if node.Type == "tree" {
				opt := gitlab.ListTreeOptions{Path: gitlab.String(node.Path), Ref: options.Ref}
				vtree, resp, err := client.recursiveListTree(pid, &opt)
				if err != nil {
					return tree, resp, err
				}
				tree = append(tree, vtree...)
			} else {
				tree = append(tree, node)
			}
		}
		if resp.CurrentPage >= resp.TotalPages {
			return tree, resp, nil
		}
		options.Page = resp.NextPage
	}

}

func (client *GitlabClient) processProject(wg *sync.WaitGroup, projectID int, checker *Checker) {
	defer wg.Done()
	project := client.getProject(projectID)
	if project == nil || project.EmptyRepo {
		return
	}

	opt := &gitlab.ListTreeOptions{Ref: gitlab.String(project.DefaultBranch), Recursive: gitlab.Bool(true)}
	tree, resp, err := client.recursiveListTree(project.ID, opt)
	if resp != nil && resp.StatusCode == 404 {
		logrus.Print(err)
	}
	if err != nil {
		logrus.Print(err)
		return
	}

	fopt := &gitlab.GetFileOptions{Ref: gitlab.String(project.DefaultBranch)}
	filesToProcess := make([]MatchFile, 0)
	for _, node := range tree {
		if checker.checkFileExtBlacklisted(node.Path) {
			continue
		}
		if checker.checkFilenameBlacklisted(node.Name) {
			continue
		}
		var file *gitlab.File
		for {
			file, resp, err = client.RepositoryFiles.GetFile(project.ID, node.Path, fopt)
			if !isTimeout(err) {
				break
			}
			time.Sleep(5 * time.Second)
		}
		if resp != nil && resp.StatusCode == 404 {
			logrus.Infof("[%s]. File not found. %s at %s", project.Name, node.Path, project.DefaultBranch)
			continue
		}

		var content []byte
		if file != nil && file.Content != "" {
			content, err = base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				logrus.Error(err)
			}
		}

		filesToProcess = append(filesToProcess, newMatchFile(node.Path, content, client.withCommit(project.ID, file.CommitID), checker))
	}
	client.CheckMatch(project, filesToProcess, checker.storage)
}

func (client *GitlabClient) withCommit(pid int, commitID string) *CommitInfo {
	commit, _, err := client.Commits.GetCommit(pid, commitID)
	if err != nil {
		logrus.Println(err)
		return &CommitInfo{}
	}
	commitInfo := &CommitInfo{
		Author:  fmt.Sprintf("%s (%s)", commit.AuthorName, commit.AuthorEmail),
		Date:    commit.CreatedAt.Format("02.01.2006 15:04:05"),
		Message: commit.Message,
		Sha:     commit.ID,
	}
	return commitInfo
}
