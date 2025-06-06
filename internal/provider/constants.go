package provider

const hierarchyDescription = `
The level of the work type in the hierarchy. There are a few rules:
	* A value of -1 indicates the work type is a subtask.
	* A value of 0 indicates the work type is a standard level work type.
	* Epics have a hierarchy level of 1, and in Jira Premium, hierarchy can be expand beyond the epic level. In this case values above 0 and below -1 can be used.`
