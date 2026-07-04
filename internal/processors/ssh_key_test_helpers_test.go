package processors

// githubKeyResponse is used by test helpers to mock GitHub API responses.
type githubKeyResponse struct {
	Key       string `json:"key"`
	ID        int    `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	Verified  bool   `json:"verified"`
	ReadOnly  bool   `json:"read_only"`
}
