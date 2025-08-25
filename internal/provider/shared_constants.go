// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"maps"
	"slices"
)

const hierarchyDescription = `
The level of the work type in the Jira issue type hierarchy:
  - -1: Sub-task (child issue type)
  - 0: Standard issue type (default level)
  - 1: Epic (Epic level)
Higher levels (2+) are available only in Jira Software Premium via Advanced Roadmaps custom hierarchy. Standard editions do not support setting levels above 0 (except -1 for sub-tasks).
References:
- Atlassian: Issue type hierarchy — https://support.atlassian.com/jira-software-cloud/docs/issue-type-hierarchy/
- Atlassian: Configure issue type hierarchy (Advanced Roadmaps) — https://support.atlassian.com/jira-software-cloud/docs/configure-issue-type-hierarchy/
`

// FieldTypeSpec represents the type specification for a field with its associated value and searcher key.
type FieldTypeSpec struct {
	Value       string
	SearcherKey string
}

// fieldTypesMap defines a mapping of custom field types to their corresponding specifications and searcher keys.
var fieldTypesMap = map[string]FieldTypeSpec{
	"cascadingselect": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:cascadingselect",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:cascadingselectsearcher",
	},
	"datepicker": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:datepicker",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:daterange",
	},
	"datetime": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:datetime",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:datetimerange",
	},
	"float": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:float",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:numberrange",
	},
	"grouppicker": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:grouppicker",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:grouppickersearcher",
	},
	"labels": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:labels",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:labelsearcher",
	},
	"multicheckboxes": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:multiselectsearcher",
	},
	"multigrouppicker": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:multigrouppicker",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:grouppickersearcher",
	},
	"multiselect": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:multiselect",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:multiselectsearcher",
	},
	"multiuserpicker": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:multiuserpicker",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:userpickergroupsearcher",
	},
	"multiversion": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:multiversion",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:versionsearcher",
	},
	"project": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:project",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:projectsearcher",
	},
	"radiobuttons": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:radiobuttons",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:multiselectsearcher",
	},
	"readonlyfield": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:readonlyfield",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:textsearcher",
	},
	"select": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:select",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:multiselectsearcher",
	},
	"textarea": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:textarea",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:textsearcher",
	},
	"textfield": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:textfield",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:textsearcher",
	},
	"url": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:url",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:exacttextsearcher",
	},
	"userpicker": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:userpicker",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:userpickergroupsearcher",
	},
	"version": {
		Value:       "com.atlassian.jira.plugin.system.customfieldtypes:version",
		SearcherKey: "com.atlassian.jira.plugin.system.customfieldtypes:versionsearcher",
	},
}

// Get sorted field type keys for custom field type enumeration.
var fieldTypeKeys = slices.Sorted(maps.Keys(fieldTypesMap))
