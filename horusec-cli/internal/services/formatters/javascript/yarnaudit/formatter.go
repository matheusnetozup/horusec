// Copyright 2020 ZUP IT SERVICOS EM TECNOLOGIA E INOVACAO SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yarnaudit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ZupIT/horusec/horusec-cli/internal/services/formatters/javascript/npmaudit"
	"os"
	"strconv"
	"strings"

	"github.com/ZupIT/horusec/development-kit/pkg/entities/analyser/javascript/yarn"
	"github.com/ZupIT/horusec/development-kit/pkg/entities/horusec"
	"github.com/ZupIT/horusec/development-kit/pkg/enums/languages"
	"github.com/ZupIT/horusec/development-kit/pkg/enums/tools"
	"github.com/ZupIT/horusec/development-kit/pkg/utils/logger"
	dockerEntities "github.com/ZupIT/horusec/horusec-cli/internal/entities/docker"
	"github.com/ZupIT/horusec/horusec-cli/internal/helpers/messages"
	"github.com/ZupIT/horusec/horusec-cli/internal/services/formatters"
)

type Formatter struct {
	formatters.IService
}

func NewFormatter(service formatters.IService) formatters.IFormatter {
	return &Formatter{
		service,
	}
}

func (f *Formatter) StartAnalysis(projectSubPath string) {
	err := f.startYarnAuditAnalysis(projectSubPath)
	f.LogAnalysisError(err, tools.YarnAudit, projectSubPath)
	f.SetLanguageIsFinished()
}

func (f *Formatter) startYarnAuditAnalysis(projectSubPath string) error {
	f.LogDebugWithReplace(messages.MsgDebugToolStartAnalysis, tools.YarnAudit)

	output, err := f.ExecuteContainer(f.getConfigDataYarn(projectSubPath))
	if err != nil {
		f.SetAnalysisError(err)
		return err
	}

	f.LogDebugWithReplace(messages.MsgDebugToolFinishAnalysis, tools.YarnAudit)
	return f.parseOutput(output)
}

func (f *Formatter) parseOutput(containerOutput string) error {
	if f.VerifyErrors(containerOutput) {
		return nil
	}

	output, err := f.newContainerOutputFromString(containerOutput)
	if err != nil {
		return err
	}

	f.processOutput(output)
	return nil
}

func (f *Formatter) newContainerOutputFromString(containerOutput string) (output *yarn.Output, err error) {
	if containerOutput == "" {
		logger.LogDebugWithLevel(messages.MsgDebugOutputEmpty, logger.DebugLevel,
			map[string]interface{}{"tool": tools.YarnAudit.ToString()})
		return &yarn.Output{}, nil
	}
	if err = json.Unmarshal([]byte(containerOutput), &output); err != nil {
		logger.LogErrorWithLevel(f.GetAnalysisIDErrorMessage(tools.YarnAudit, containerOutput),
			err, logger.ErrorLevel)
	}

	return output, err
}

func (f *Formatter) setVulnerabilitySeverityData(output *yarn.Issue) *horusec.Vulnerability {
	data := f.getDefaultVulnerabilitySeverity()
	data.Severity = output.GetSeverity()
	data.Version = output.GetVersion()
	data.Details = output.Overview
	data.Code = output.ModuleName
	data.VulnerableBelow = output.VulnerableVersions
	data.Line = f.getVulnerabilityLineByName(data.Code, data.Version, data.File)
	return f.setCommitAuthor(data)
}

func (f *Formatter) setCommitAuthor(vulnerability *horusec.Vulnerability) *horusec.Vulnerability {
	commitAuthor := f.GetCommitAuthor(vulnerability.Line, vulnerability.File)

	vulnerability.CommitAuthor = commitAuthor.Author
	vulnerability.CommitHash = commitAuthor.CommitHash
	vulnerability.CommitDate = commitAuthor.Date
	vulnerability.CommitEmail = commitAuthor.Email
	vulnerability.CommitMessage = commitAuthor.Message

	return vulnerability
}

func (f *Formatter) getDefaultVulnerabilitySeverity() *horusec.Vulnerability {
	vulnerabilitySeverity := &horusec.Vulnerability{}
	vulnerabilitySeverity.SecurityTool = tools.YarnAudit
	vulnerabilitySeverity.Language = languages.Javascript
	vulnerabilitySeverity.File = "yarn.lock"
	return vulnerabilitySeverity
}

func (f *Formatter) IsNotFoundError(containerOutput string) bool {
	return strings.Contains(containerOutput, "ERROR_YARN_LOCK_NOT_FOUND")
}

func (f *Formatter) IsRunningError(containerOutput string) bool {
	return strings.Contains(containerOutput, "ERROR_RUNNING_YARN_AUDIT")
}

func (f *Formatter) setNotFoundError() {
	f.SetAnalysisError(errors.New(messages.MsgErrorYarnLockNotFound))
}

func (f *Formatter) setRunningError(containerOutput string) {
	f.SetAnalysisError(errors.New(messages.MsgErrorYarnProcess + containerOutput))
}

func (f *Formatter) VerifyErrors(containerOutput string) bool {
	if f.IsNotFoundError(containerOutput) {
		f.setNotFoundError()
		return true
	}

	if f.IsRunningError(containerOutput) {
		f.setRunningError(containerOutput)
		return true
	}

	return false
}

func (f *Formatter) processOutput(output *yarn.Output) {
	for _, advisory := range output.Advisories {
		value := advisory
		vulnerability := f.setVulnerabilitySeverityData(&value)
		f.GetAnalysis().Vulnerabilities = append(f.GetAnalysis().Vulnerabilities, *vulnerability)
	}
}

func (f *Formatter) getVulnerabilityLineByName(module, version, file string) string {
	path := fmt.Sprintf("%s/%s", f.GetConfigProjectPath(), file)
	fileExisting, err := os.Open(path)
	if err != nil {
		return ""
	}

	defer func() {
		logger.LogErrorWithLevel(messages.MsgErrorDeferFileClose, fileExisting.Close(), logger.ErrorLevel)
	}()
	scanner := bufio.NewScanner(fileExisting)
	return f.getLine(module, version, scanner)
}

func (f *Formatter) getLine(module, version string, scanner *bufio.Scanner) string {
	line := 1
	for scanner.Scan() {
		if f.validateIfExistNameInScannerText(scanner.Text(), module, version) {
			return strconv.Itoa(line)
		}
		line++
	}
	return ""
}

func (f *Formatter) validateIfExistNameInScannerText(
	scannerText, module, version string) bool {
	for _, name := range f.mapPossibleExistingNames(module, version) {
		if strings.Contains(strings.ToLower(scannerText), name) {
			return true
		}
	}
	return false
}

func (f *Formatter) mapPossibleExistingNames(module, version string) []string {
	return []string{
		strings.ToLower(fmt.Sprintf("%s@%s", module, version)),
		strings.ToLower(fmt.Sprintf("%s@~%s", module, version)),
		strings.ToLower(fmt.Sprintf("%s@^%s", module, version)),
	}
}

func (f *Formatter) getConfigDataYarn(projectSubPath string) *dockerEntities.AnalysisData {
	return &dockerEntities.AnalysisData{
		Image:    npmaudit.ImageName,
		Tag:      npmaudit.ImageTag,
		CMD:      f.AddWorkDirInCmd(ImageCmd, projectSubPath),
		Language: languages.Javascript,
	}
}
