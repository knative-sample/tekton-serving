package trigger

import (
	"encoding/json"
	"net/http"
	"os"

	"bytes"
	"strings"

	"fmt"

	"io/ioutil"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/golang/glog"
	gh "gopkg.in/go-playground/webhooks.v5/github"
)

const (
	ApplicationJSON = "application/json"

	// gitHubEventTypePrefix is what all GitHub event types get
	// prefixed with when converting to CloudEvents.
	gitHubEventTypePrefix = "dev.knative.source.github"
)

// GitHubEventType returns the GitHub CloudEvent type value.
func gitHubEventType(ghEventType gh.Event) string {
	return fmt.Sprintf("%s.%s", gitHubEventTypePrefix, ghEventType)
}

type Trigger struct {
	TriggerConfig string
}

type Args struct {
	ShortCommitid string
	Commitid      string
	Branch        string
	TimeString    string
}

func (dp *Trigger) Run() error {
	glog.Info("Trigger is run")

	http.HandleFunc("/api/event", dp.Event)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	return nil
}

func (api *Trigger) Event(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(body)
}
func (dp *Trigger) run(e cloudevents.Event) error {
	switch e.Context.GetType() {
	case gitHubEventType(gh.PingEvent):
		dp.logEvent(e)
	case gitHubEventType(gh.PullRequestEvent):
		return dp.pullRequestMergedEvent(e)
	default:
		glog.Infof("ingore Event: %s ", e.Context.GetType())
	}

	return nil
}

func (dp *Trigger) logEvent(e cloudevents.Event) {
	b := strings.Builder{}
	if e.Data != nil {
		b.WriteString("Data,\n  ")
		if strings.HasPrefix(e.DataContentType(), ApplicationJSON) {
			var prettyJSON bytes.Buffer

			data, ok := e.Data.([]byte)
			if !ok {
				var err error
				data, err = json.Marshal(e.Data)
				if err != nil {
					data = []byte(err.Error())
				}
			}
			err := json.Indent(&prettyJSON, data, "  ", "  ")
			if err != nil {
				b.Write(e.Data.([]byte))
			} else {
				b.Write(prettyJSON.Bytes())
			}
		} else {
			b.Write(e.Data.([]byte))
		}
		b.WriteString("\n")
	}

	glog.Infof("cloudevents.Event\n  Type:%s\n  Data:%s", e.Context.GetType(), b.String())
}
