package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type GitlabClient struct {
	*gitlab.Client
	accessToken *AccessToken
}

func (client *GitlabClient) CheckMatch(ctx context.Context, project *gitlab.Project, branchName string, filesToProcess []MatchFile) {
	var (
		records []MatchRecord
	)
	messages := getContextMessageStore(ctx).getMessages(project.NameWithNamespace)
	for _, fileToMatch := range filesToProcess {
		if record, reason, matched := getMatch(ctx, fileToMatch); matched {
			records = append(records, record)
			msg := &Message{
				MatchType:       record.MatchType,
				ProjectID:       project.ID,
				ProjectName:     project.Namespace.Name + "/" + project.Name,
				MatchURL:        project.WebURL + "/-/blob/" + branchName + "/" + fileToMatch.Path,
				Path:            fileToMatch.Path,
				Filename:        record.MatchedFilename,
				RawMatchContent: record.MatchedContent,
				MatchName:       reason,
				Confidence:      record.Confidence,
				commitInfo:      fileToMatch.CommitInfo,
			}
			logrus.Printf("Got match in %s. Line: %d. Path: %s. Reason: %s", fileToMatch.Filename, record.MatchedLineNumbers, msg.MatchURL, msg.MatchName)
			messages = append(messages, msg)
		}
	}
	session := getContextScanSession(ctx)
	session.add(project.NameWithNamespace + "_" + strings.ToLower(branchName))
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

func (client *GitlabClient) isProjectActive(ctx context.Context, projectID int) bool {
	optMonthToCheckFrom := getContextOptions(ctx).FromMonth
	timeDelta := gitlab.ISOTime(time.Now().AddDate(0, -optMonthToCheckFrom, 0))

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

	return !(len(projectEvents) == 0)
}

func (client *GitlabClient) createTasksAndWait(ctx context.Context, projectsChan chan int) (wait <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		var tasksGroup sync.WaitGroup
		projectsCount := getContextOptions(ctx).ProjectsCount
		for i := 0; i < projectsCount; i++ {
			tasksGroup.Add(1)
			go client.task(ctx, &tasksGroup, projectsChan)
		}

		tasksGroup.Wait()
		close(ch)
	}()
	return ch
}

func (client *GitlabClient) task(ctx context.Context, wg *sync.WaitGroup, projectsChan chan int) {
	defer wg.Done()
	for projectID := range projectsChan {
		if client.isProjectActive(ctx, projectID) {
			logrus.Debugf("task for processing project with pid = %d\n", projectID)
			client.processProject(ctx, projectID)
		}
	}
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

func (client *GitlabClient) processProjectBranch(ctx context.Context, wg *sync.WaitGroup, project *gitlab.Project, branchNameChan <-chan string) {
	defer wg.Done()
	session := getContextScanSession(ctx)
	for branchName := range branchNameChan {
		logrus.Debugf("[%s] Processing branch %s", project.NameWithNamespace, branchName)
		if session.check(project.NameWithNamespace + "_" + strings.ToLower(branchName)) {
			logrus.Debugf("%s at %s branch has already been scanned.", project.NameWithNamespace, branchName)
			continue
		}
		opt := &gitlab.ListTreeOptions{Ref: gitlab.String(branchName), Recursive: gitlab.Bool(true)}
		tree, resp, err := client.recursiveListTree(project.ID, opt)
		if resp != nil && resp.StatusCode == 404 {
			logrus.Print(err)
		}
		if err != nil {
			logrus.Print(err)
			return
		}

		fopt := &gitlab.GetFileOptions{Ref: gitlab.String(branchName)}
		filesToProcess := make([]MatchFile, 0)
		for _, node := range tree {
			if checkFileExtBlacklisted(ctx, node.Path) {
				continue
			}
			if checkFilenameBlacklisted(ctx, node.Name) {
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
				logrus.Debugf("[%s]. File not found. %s at %s", project.Name, node.Path, project.DefaultBranch)
				continue
			}

			var content []byte
			if file != nil && file.Content != "" {
				content, err = base64.StdEncoding.DecodeString(file.Content)
				if err != nil {
					logrus.Error(err)
				}
			}

			filesToProcess = append(filesToProcess, newMatchFile(node.Path, content, client.withCommit(project.ID, file)))
		}
		client.CheckMatch(ctx, project, branchName, filesToProcess)
	}
}

func (client *GitlabClient) processProject(ctx context.Context, projectID int) {
	project := client.getProject(projectID)
	if project == nil || project.EmptyRepo {
		return
	}
	bopts := &gitlab.ListBranchesOptions{}
	branches := make([]*gitlab.Branch, 0)
	for {
		br, resp, err := client.Branches.ListBranches(projectID, bopts)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		if err != nil {
			logrus.Errorln(err)
		}
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		bopts.Page = resp.NextPage
		branches = append(branches, br...)
	}
	routineCount := 1
	branchNameChan := make(chan string, 0)
	var pbWg sync.WaitGroup
	for i := 0; i < routineCount; i++ {
		pbWg.Add(1)
		go client.processProjectBranch(ctx, &pbWg, project, branchNameChan)
	}
	for _, branch := range branches {
		branchNameChan <- branch.Name
	}
	pbWg.Wait()
}

func (client *GitlabClient) withCommit(pid int, file *gitlab.File) *CommitInfo {
	if file == nil {
		return &CommitInfo{}
	}
	commit, _, err := client.Commits.GetCommit(pid, file.CommitID)
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
