// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package testhelpers

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// TemplatePath returns an absolute path to a template file under testdata/templates.
func TemplatePath(name string) string {
	return filepath.Join("testdata", "templates", name)
}

// MustReadTemplate reads a template by name or fails the test.
func MustReadTemplate(t *testing.T, name string) string {
	t.Helper()
	p := TemplatePath(name)
	absPath, _ := filepath.Abs(p)
	b, err := os.ReadFile(p)
	if err != nil {
		wd, _ := os.Getwd()
		dir := filepath.Dir(p)
		var candidates []string
		if entries, dirErr := os.ReadDir(dir); dirErr == nil {
			for _, e := range entries {
				if !e.IsDir() {
					candidates = append(candidates, e.Name())
				}
			}
		}
		t.Fatalf(
			"failed to read template %q\n  path: %s\n  abs:  %s\n  cwd:  %s\n  dir:  %s\n  available templates: %v\n  error: %v",
			name, p, absPath, wd, dir, candidates, err,
		)
	}
	return string(b)
}

// Common hierarchy levels used across tests.
const (
	HierarchyStandard = 0
	HierarchySubtask  = -1
)

// MustCopy copies from r to a temp file and returns its path.
func MustCopy(t *testing.T, name string, r io.Reader) string {
	t.Helper()
	tmp := t.TempDir()
	dst := filepath.Join(tmp, name)
	f, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return dst
}

// BuildLargeBody creates a large JSON-like string embedding various secrets
// to validate both truncation and redaction. Size target ~2MB.
func BuildLargeBody() string {
	var b strings.Builder
	// approx 2MB total
	chunks := 2 << 20 / 64
	for i := 0; i < chunks; i++ {
		b.WriteString(`{"authorization":"Bearer TOPSECRET`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`","access_token":"AAA`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`","password":"PWD`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`"}`)
	}
	return b.String()
}

func MkRSWithBodyAndHeaders(code int, hdr http.Header, body string) *models.ResponseScheme {
	var buf bytes.Buffer
	buf.WriteString(body)
	rs := &models.ResponseScheme{Code: code, Response: &http.Response{StatusCode: code, Header: http.Header{}}, Bytes: buf}
	for k, v := range hdr {
		rs.Header[k] = v
	}
	return rs
}

// MkRS helper to build a models.ResponseScheme with convenience fields
func MkRS(code int, headers http.Header, body string) *models.ResponseScheme {
	var buf bytes.Buffer
	if body != "" {
		buf.WriteString(body)
	}
	rs := &models.ResponseScheme{
		Code:     code,
		Response: &http.Response{StatusCode: code, Header: http.Header{}},
		Bytes:    buf,
	}

	for k, v := range headers {
		rs.Header[k] = v
	}

	return rs
}

// TestAccProjectWithDataSource generates a Terraform configuration string for creating a Jira project and data sources.
// This function requires project attributes such as key, name, project type, and lookup type.
// It uses a template file to build the configuration and executes it with provided parameters.
// The returned string can be used in acceptance tests to verify resource and data source integration.
func TestAccProjectWithDataSource(t *testing.T, key, name, projectType, lookupType string) string {
	t.Helper()

	projectResTmpl, err := template.New(ProjectTmpl).ParseFiles(ProjectTmplPath)

	if err != nil {
		t.Fatal(err)
	}

	var projTF bytes.Buffer

	projectTmplCfg := ProjectTmplCfg{
		Key:           key,
		Name:          name,
		ProjectType:   projectType,
		LeadAccountID: TestAccLeadAccountID(),
	}

	if err = projectResTmpl.Execute(&projTF, projectTmplCfg); err != nil {
		t.Fatal(err)
	}

	dataTmpl, err := template.New(DataProjectTmpl).ParseFiles(DataProjectTmplPath)
	if err != nil {
		t.Fatal(err)
	}

	var dataTf bytes.Buffer
	dataName := "by_key"
	if lookupType == "id" {
		dataName = "by_id"
	}

	dataTmplCfg := DataProjectsCfg{
		ProjectResources: []string{projTF.String()},
		DataName:         dataName,
		LookupBy:         lookupType,
	}
	if err = dataTmpl.Execute(&dataTf, dataTmplCfg); err != nil {
		t.Fatal(err)
	}

	return dataTf.String()
}

func TestAccLeadAccountID() string {
	return strings.TrimSpace(os.Getenv("JIRA_PROJECT_TEST_ROLE_LEAD_ID"))
}

func TestAccProjectResourceConfig(t *testing.T, key, name, projectType, leadAccountID, description string) string {
	t.Helper()

	tmpl, err := template.New(ProjectTmpl).ParseFiles(ProjectTmplPath)

	if err != nil {
		t.Fatal(err)
	}

	var projectTf bytes.Buffer

	projectTmplCfg := ProjectTmplCfg{
		Key:           key,
		Name:          name,
		ProjectType:   projectType,
		LeadAccountID: leadAccountID,
		Description:   description,
	}

	err = tmpl.Execute(&projectTf, projectTmplCfg)

	if err != nil {
		t.Fatal(err)
	}

	return projectTf.String()
}
