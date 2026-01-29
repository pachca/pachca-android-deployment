package shared

type ReleaseInfo struct {
	JobID       int    `json:"job_id"`
	VersionCode int    `json:"version_code"`
	VersionName string `json:"version_name"`
}

const (
	EnvPachcaUrl string = "ENV_PACHCA_URL"
	EnvGitlabUrl string = "ENV_GITLAB_URL"
	EnvLinearUrl string = "ENV_LINEAR_URL"

	EnvPachcaKey string = "ENV_PACHCA_KEY"
	EnvGitlabKey string = "ENV_GITLAB_KEY"
	EnvLinearKey string = "ENV_LINEAR_KEY"

	EnvPachcaInternalChatId string = "ENV_PACHCA_INTERNAL_CHAT_ID"
	EnvPachcaPublicChatId   string = "ENV_PACHCA_PUBLIC_CHAT_ID"

	EnvLinearTeamId string = "ENV_LINEAR_TEAM_ID"
)
