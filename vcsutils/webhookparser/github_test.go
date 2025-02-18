package webhookparser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/jfrog/froggit-go/vcsclient"
	"github.com/jfrog/froggit-go/vcsutils"
)

const (
	githubSha256Header = "X-HUB-SIGNATURE-256"
	githubEventHeader  = "X-GITHUB-EVENT"

	// Push event
	githubPushSha256       = "687737b6d39345e557be42058da1ad57dbd5f54baeb30044751e50d396cc2116"
	githubPushExpectedTime = int64(1630416256)
	// Pull request create event
	githubPrOpenSha256         = "48b9f23bfeb95dd8a1067b590f753599dbd12732c8d5217431ec70132cee8c1c"
	githubPrOpenExpectedTime   = int64(1630666350)
	githubPrReopenSha256       = "07021461375abc8e232f63f68bffc8630898b68a1e112e402c7b658e5245b791"
	githubPrReopenExpectedTime = int64(1638805321)
	// Pull request update event
	githubPrSyncSha256       = "4e03348ae316bae2719c0e936b55535e7869a85aa5649021ea5055b378cd6d56"
	githubPrSyncExpectedTime = int64(1630666481)
	githubPrEditSha256       = "b1ee2dfd35d9eac32374e4a0aa6bbee1752c0d19640295bc29a793971999be29"
	githubPrEditExpectedTime = int64(1638802767)
	// Pull request close event
	githubPrCloseSha256       = "51cddb70352880cfd2a8ba2b55d3e5ed827b8ca528a6dc31e346a5b4d3485496"
	githubPrCloseExpectedTime = int64(1638804604)
	// Pull request merge event
	githubPrMergeSha256       = "f94088bf7c34740ed9f9c3752f30e786527fbe5f5c9726d4526d9c92b5a7c208"
	githubPrMergeExpectedTime = int64(1638805994)
	gitHubExpectedPrID        = 2
)

func TestGitHubParseIncomingPushWebhook(t *testing.T) {
	reader, err := os.Open(filepath.Join("testdata", "github", "pushpayload"))
	require.NoError(t, err)
	defer close(reader)

	// Create request
	request := httptest.NewRequest("POST", "https://127.0.0.1", reader)
	request.Header.Add("content-type", "application/x-www-form-urlencoded")
	request.Header.Add(githubSha256Header, "sha256="+githubPushSha256)
	request.Header.Add(githubEventHeader, "push")

	// Parse webhook
	actual, err := ParseIncomingWebhook(context.Background(),
		vcsclient.EmptyLogger{},
		WebhookOrigin{
			VcsProvider: vcsutils.GitHub,
			Token:       token,
		}, request)
	require.NoError(t, err)

	// Check values
	assert.Equal(t, expectedRepoName, actual.TargetRepositoryDetails.Name)
	assert.Equal(t, expectedOwner, actual.TargetRepositoryDetails.Owner)
	assert.Equal(t, expectedBranch, actual.TargetBranch)
	assert.Equal(t, githubPushExpectedTime, actual.Timestamp)
	assert.Equal(t, vcsutils.Push, actual.Event)
	assert.Equal(t, WebHookInfoUser{Login: "yahavi", DisplayName: "Yahav Itzhak", Email: "yahavi@users.noreply.github.com"}, actual.Author)
	assert.Equal(t, WebHookInfoUser{Login: "web-flow", DisplayName: "GitHub", Email: "noreply@github.com"}, actual.Committer)
	assert.Equal(t, WebHookInfoUser{DisplayName: "yahavi", Email: "yahavi@users.noreply.github.com"}, actual.TriggeredBy)
	assert.Equal(t, WebHookInfoCommit{
		Hash:    "9d497bd67a395a8063774f200338769ccbcee916",
		Message: "Update README.md",
		Url:     "https://github.com/yahavi/hello-world/commit/9d497bd67a395a8063774f200338769ccbcee916",
	}, actual.Commit)
	assert.Equal(t, WebHookInfoCommit{
		Hash: "a82aa1b065b4fa17db4b7a055109044be377ddf7",
	}, actual.BeforeCommit)
	assert.Equal(t, WebhookInfoBranchStatusUpdated, actual.BranchStatus)
	assert.Equal(t, "https://github.com/yahavi/hello-world/compare/a82aa1b065b4fa17db4b7a055109044be377ddf7...9d497bd67a395a8063774f200338769ccbcee916", actual.CompareUrl)
}

func TestGithubParseIncomingPrWebhook(t *testing.T) {
	tests := []struct {
		name              string
		payloadFilename   string
		payloadSha        string
		expectedTime      int64
		expectedEventType vcsutils.WebhookEvent
	}{
		{
			name:              "open",
			payloadFilename:   "propenpayload",
			payloadSha:        githubPrOpenSha256,
			expectedTime:      githubPrOpenExpectedTime,
			expectedEventType: vcsutils.PrOpened,
		},
		{
			name:              "reopen",
			payloadFilename:   "prreopenpayload",
			payloadSha:        githubPrReopenSha256,
			expectedTime:      githubPrReopenExpectedTime,
			expectedEventType: vcsutils.PrOpened,
		},
		{
			name:              "synchronize",
			payloadFilename:   "prsynchronizepayload",
			payloadSha:        githubPrSyncSha256,
			expectedTime:      githubPrSyncExpectedTime,
			expectedEventType: vcsutils.PrEdited,
		},
		{
			name:              "edit",
			payloadFilename:   "preditpayload",
			payloadSha:        githubPrEditSha256,
			expectedTime:      githubPrEditExpectedTime,
			expectedEventType: vcsutils.PrEdited,
		},
		{
			name:              "close",
			payloadFilename:   "prclosepayload",
			payloadSha:        githubPrCloseSha256,
			expectedTime:      githubPrCloseExpectedTime,
			expectedEventType: vcsutils.PrRejected,
		},
		{
			name:              "merge",
			payloadFilename:   "prmergepayload",
			payloadSha:        githubPrMergeSha256,
			expectedTime:      githubPrMergeExpectedTime,
			expectedEventType: vcsutils.PrMerged,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := os.Open(filepath.Join("testdata", "github", tt.payloadFilename))
			require.NoError(t, err)
			defer close(reader)

			// Create request
			request := httptest.NewRequest("POST", "https://127.0.0.1", reader)
			request.Header.Add("content-type", "application/x-www-form-urlencoded")
			request.Header.Add(githubSha256Header, "sha256="+tt.payloadSha)
			request.Header.Add(githubEventHeader, "pull_request")

			// Parse webhook
			actual, err := ParseIncomingWebhook(context.Background(),
				vcsclient.EmptyLogger{},
				WebhookOrigin{
					VcsProvider: vcsutils.GitHub,
					Token:       token,
				}, request)
			require.NoError(t, err)

			// Check values
			assert.Equal(t, gitHubExpectedPrID, actual.PullRequestId)
			assert.Equal(t, expectedRepoName, actual.TargetRepositoryDetails.Name)
			assert.Equal(t, expectedOwner, actual.TargetRepositoryDetails.Owner)
			assert.Equal(t, expectedBranch, actual.TargetBranch)
			assert.Equal(t, tt.expectedTime, actual.Timestamp)
			assert.Equal(t, expectedRepoName, actual.SourceRepositoryDetails.Name)
			assert.Equal(t, expectedOwner, actual.SourceRepositoryDetails.Owner)
			assert.Equal(t, expectedSourceBranch, actual.SourceBranch)
			assert.Equal(t, tt.expectedEventType, actual.Event)
		})
	}
}

func TestGitHubParseIncomingWebhookError(t *testing.T) {
	request := &http.Request{}
	_, err := ParseIncomingWebhook(context.Background(),
		vcsclient.EmptyLogger{},
		WebhookOrigin{
			VcsProvider: vcsutils.GitHub,
			Token:       token,
		}, request)

	require.Error(t, err)

	webhook := gitHubWebhookParser{logger: vcsclient.EmptyLogger{}}
	_, err = webhook.parseIncomingWebhook(context.Background(), request, []byte{})
	assert.Error(t, err)
}

func TestGitHubParsePrEventsError(t *testing.T) {
	webhook := gitHubWebhookParser{}
	assert.Nil(t, webhook.parsePrEvents(nil))
}

func TestGitHubPayloadMismatchSignature(t *testing.T) {
	reader, err := os.Open(filepath.Join("testdata", "github", "pushpayload"))
	require.NoError(t, err)
	defer close(reader)

	// Create request
	request := httptest.NewRequest("POST", "https://127.0.0.1", reader)
	request.Header.Add("content-type", "application/x-www-form-urlencoded")
	request.Header.Add(githubSha256Header, "sha256=wrongsignature")
	request.Header.Add(githubEventHeader, "push")

	// Parse webhook
	_, err = ParseIncomingWebhook(context.Background(),
		vcsclient.EmptyLogger{},
		WebhookOrigin{
			VcsProvider: vcsutils.GitHub,
			Token:       token,
		}, request)
	assert.True(t, strings.HasPrefix(err.Error(), "error decoding signature"), "error was: "+err.Error())
}
