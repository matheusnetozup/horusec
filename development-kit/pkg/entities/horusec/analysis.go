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

package horusec

import (
	"encoding/json"
	"time"

	"github.com/ZupIT/horusec/development-kit/pkg/enums/severity"

	"github.com/ZupIT/horusec/development-kit/pkg/enums/horusec"
	"github.com/google/uuid"
)

type Analysis struct {
	ID              uuid.UUID       `json:"id" gorm:"Column:analysis_id"`
	RepositoryID    uuid.UUID       `json:"repositoryID" gorm:"Column:repository_id"`
	RepositoryName  string          `json:"repositoryName" gorm:"Column:repository_name"`
	CompanyID       uuid.UUID       `json:"companyID" gorm:"Column:company_id"`
	CompanyName     string          `json:"companyName" gorm:"Column:company_name"`
	Status          horusec.Status  `json:"status" gorm:"Column:status"`
	Errors          string          `json:"errors" gorm:"Column:errors"`
	CreatedAt       time.Time       `json:"createdAt" gorm:"Column:created_at"`
	FinishedAt      time.Time       `json:"finishedAt" gorm:"Column:finished_at"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities" gorm:"foreignkey:AnalysisID;association_foreignkey:ID"`
}

func (a *Analysis) GetTable() string {
	return "analysis"
}

func (a *Analysis) ToBytes() []byte {
	bytes, _ := json.Marshal(a)
	return bytes
}

func (a *Analysis) GetID() uuid.UUID {
	return a.ID
}

func (a *Analysis) GetIDString() string {
	return a.ID.String()
}

func (a *Analysis) ToString() string {
	return string(a.ToBytes())
}

func (a *Analysis) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":              a.ID,
		"createdAt":       a.CreatedAt,
		"repositoryID":    a.RepositoryID,
		"repositoryName":  a.RepositoryName,
		"companyName":     a.CompanyName,
		"companyID":       a.CompanyID,
		"status":          a.Status,
		"errors":          a.Errors,
		"finishedAt":      a.FinishedAt,
		"vulnerabilities": a.Vulnerabilities,
	}
}

func (a *Analysis) SetFindOneFilter() map[string]interface{} {
	return map[string]interface{}{"id": a.GetID()}
}

func (a *Analysis) SetAnalysisError(err error) {
	if err != nil {
		toAppend := ""
		if len(a.Errors) > 0 {
			a.Errors += ", " + err.Error()
			return
		}
		a.Errors += toAppend + err.Error()
	}
}

func (a *Analysis) GenerateID() {
	a.ID = uuid.New()
}

func (a *Analysis) SetCompanyName(companyName string) {
	a.CompanyName = companyName
}

func (a *Analysis) SetRepositoryName(repositoryName string) {
	a.RepositoryName = repositoryName
}

func (a *Analysis) SetRepositoryID(repositoryID uuid.UUID) {
	a.RepositoryID = repositoryID
}

func (a *Analysis) SetAnalysisIDAndNewIDInVulnerabilities() {
	for key := range a.Vulnerabilities {
		a.Vulnerabilities[key].AnalysisID = a.ID
		a.Vulnerabilities[key].VulnerabilityID = uuid.New()
	}
}

func (a *Analysis) SetAnalysisFinishedData() *Analysis {
	a.FinishedAt = time.Now()

	if a.HasErrors() {
		a.Status = horusec.Error
		return a
	}

	a.Status = horusec.Success
	return a
}

func (a *Analysis) HasErrors() bool {
	return len(a.Errors) > 0
}

func (a *Analysis) GetTotalVulnerabilities() int {
	return len(a.Vulnerabilities)
}

func (a *Analysis) GetTotalVulnerabilitiesBySeverity() (total map[severity.Severity]int) {
	total = map[severity.Severity]int{
		severity.High:   0,
		severity.Medium: 0,
		severity.Low:    0,
		severity.Info:   0,
		severity.Audit:  0,
		severity.NoSec:  0,
	}
	for index := range a.Vulnerabilities {
		total[a.Vulnerabilities[index].Severity]++
	}
	return total
}

func (a *Analysis) SortVulnerabilitiesByCriticality() *Analysis {
	vulnerabilities := a.getVulnerabilitiesBySeverity(severity.High)
	vulnerabilities = append(vulnerabilities, a.getVulnerabilitiesBySeverity(severity.Medium)...)
	vulnerabilities = append(vulnerabilities, a.getVulnerabilitiesBySeverity(severity.Low)...)
	vulnerabilities = append(vulnerabilities, a.getVulnerabilitiesBySeverity(severity.Info)...)
	vulnerabilities = append(vulnerabilities, a.getVulnerabilitiesBySeverity(severity.Audit)...)
	vulnerabilities = append(vulnerabilities, a.getVulnerabilitiesBySeverity(severity.NoSec)...)
	a.Vulnerabilities = vulnerabilities
	return a
}

func (a *Analysis) getVulnerabilitiesBySeverity(search severity.Severity) (response []Vulnerability) {
	for index := range a.Vulnerabilities {
		if a.Vulnerabilities[index].Severity == search {
			response = append(response, a.Vulnerabilities[index])
		}
	}
	return response
}
