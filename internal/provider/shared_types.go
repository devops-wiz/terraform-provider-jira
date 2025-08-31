// SPDX-License-Identifier: MPL-2.0

package provider

import (
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
)

type ServiceClient struct {
	client           *jira.Client
	providerTimeouts opTimeouts
}
