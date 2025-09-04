// SPDX-License-Identifier: MPL-2.0

package constants

import (
	"maps"
	"slices"
)

// FieldTypeSpec defines the structure for representing a field type with its unique value and associated searcher key.
type FieldTypeSpec struct {
	Value       string
	SearcherKey string
}

// FieldTypesMap maps a short field type key to its corresponding FieldTypeSpec containing API value and searcher key.
var FieldTypesMap = map[string]FieldTypeSpec{
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

// FieldTypeKeys is a sorted slice of keys derived from the FieldTypesMap, representing valid field type identifiers.
var FieldTypeKeys = slices.Sorted(maps.Keys(FieldTypesMap))

// GetFieldTypeShort maps an API field type value to its corresponding short field type key from the fieldTypesMap.
func GetFieldTypeShort(apiValue string) string {
	var matchingKey string

	for key, spec := range FieldTypesMap {
		if spec.Value == apiValue {
			matchingKey = key
			break
		}
	}

	return matchingKey
}
