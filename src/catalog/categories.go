package catalog

import "strings"

type Category string

const (
	CategoryPackageManagers Category = "Package Managers"
	CategoryRuntimes        Category = "Runtimes"
	CategoryShellTerminal   Category = "Shell & Terminal"
	CategoryCLIApps         Category = "Command line"
	CategoryGit             Category = "Git"
	CategoryEditors         Category = "Editors & IDEs"
	CategoryContainersVMs   Category = "Containers & VMs"
	CategoryDeploy          Category = "Deploy"
	CategoryAIAgents        Category = "AI Agents"
	CategoryAITooling       Category = "AI Tooling"
	CategoryAIApps          Category = "AI Apps"
	CategoryBrowsers        Category = "Browsers"
	CategoryCommunication   Category = "Communication"
	CategoryProductivity    Category = "Productivity"
	CategoryMedia           Category = "Media"
	CategoryRemoteAccess    Category = "Remote Access"
	CategorySecurityApps    Category = "Security"
	CategorySystemUtilities Category = "System Utilities"
	CategoryAgentSkills     Category = "Agent skills"
)

var Categories = []Category{
	CategoryPackageManagers,
	CategoryRuntimes,
	CategoryShellTerminal,
	CategoryCLIApps,
	CategoryGit,
	CategoryEditors,
	CategoryContainersVMs,
	CategoryDeploy,
	CategoryAIAgents,
	CategoryAITooling,
	CategoryAIApps,
	CategoryBrowsers,
	CategoryCommunication,
	CategoryProductivity,
	CategoryMedia,
	CategoryRemoteAccess,
	CategorySecurityApps,
	CategorySystemUtilities,
	CategoryAgentSkills,
}

func (c Category) Folder() string { return folderOf(string(c)) }

var SettingCategories = []SettingCategory{
	CategoryShell, CategoryTerminal, CategorySecurity, CategoryEditor,
	CategorySystem, CategoryHeadless, CategoryNetwork, CategoryAgents,
}

func (c SettingCategory) Folder() string { return folderOf(string(c)) }

func folderOf(title string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(title, " & ", "-"), " ", "-"))
}

func appCategory(folder string) (Category, bool) {
	for _, c := range Categories {
		if c.Folder() == folder {
			return c, true
		}
	}
	return "", false
}

func settingCategory(folder string) (SettingCategory, bool) {
	for _, c := range SettingCategories {
		if c.Folder() == folder {
			return c, true
		}
	}
	return "", false
}
