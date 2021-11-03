package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/security-products/analyzers/report/v2"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

func readAndLocateFile(pathToFile, filename string) ([]byte, string, error) {
	var (
		data     []byte
		err      error
		fullPath string
	)

	if len(pathToFile) > 0 {
		fullPath = path.Join(pathToFile, filename)
		data, err = ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fullPath, err
		}
	} else {
		ex, err := os.Executable()
		dir := filepath.Dir(ex)
		fullPath = path.Join(dir, filename)
		data, err = ioutil.ReadFile(fullPath)
		if err != nil {
			dir, _ = os.Getwd()
			fullPath = path.Join(dir, filename)
			data, err = ioutil.ReadFile(fullPath)
			if err != nil {
				return nil, fullPath, err
			}
		}
	}
	return data, fullPath, nil
}

func getUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func gitlabIgnoreAuthority() gitlab.ClientOptionFunc {
	return func(c *gitlab.Client) error {
		gClient := getUnexportedField(reflect.ValueOf(c).Elem().FieldByName("client"))
		if gClient, ok := gClient.(*retryablehttp.Client); ok {
			gClient.HTTPClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		} else {
			return errors.New("failed to get unexported filed of client")
		}
		return nil
	}
}

func isTimeout(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}
	return false
}

func readTokenFromFile(tokenPath string) string {
	var bToken strings.Builder
	fileLocation := strings.TrimPrefix(tokenPath, "file://")
	tBytes, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		panic(err)
	}
	bToken.Write(tBytes)
	return bToken.String()
}

func saveAndSendReport(report *report.Report, projectName string, storage *Storage) (bool, error) {
	reportBytes, err := json.Marshal(report)
	if err != nil {
		return false, err
	}
	filename := path.Join(storage.getReportsDir(), projectName+"-secrets.json")
	err = ioutil.WriteFile(filename, reportBytes, 0644)
	if err != nil {
		return false, err
	}
	if DefectDodjo != nil {
		sendToDodjo(filename, (*time.Time)(report.Scan.EndTime), projectName, storage)
	}
	return true, nil
}

func sendToDodjo(filename string, endTime *time.Time, projectName string, storage *Storage) {
	tm := time.Now().Format("02.01.2006 15:05")
	engName := fmt.Sprintf("%s-engagement", projectName)
	engDesc := fmt.Sprintf("Pyromanced at %s", tm)
	startTime := endTime.Add(-time.Hour)
	defectProduct := DefectDodjo.GetProductByNameOne(storage.getDodjoProductName())
	if defectProduct == nil {
		logrus.Error("Cannot find selected product")
		return
	}
	engagement, err := defectProduct.AddEngagement(engName, engDesc, &startTime, endTime)
	if err != nil {
		logrus.Println(err)
	}
	err = engagement.ImportSecretDetectionReport(filename)
	if err != nil {
		logrus.Println(err)
	}
}

func findLineOnFile(content []byte, neededString string) int {
	scanner := bufio.NewScanner(bytes.NewBuffer(content))
	lineNum := 1
	for scanner.Scan() {
		currLine := scanner.Text()
		if strings.Contains(currLine, neededString) {
			return lineNum
		}
		lineNum++
	}
	lineNum = 0

	return lineNum
}
