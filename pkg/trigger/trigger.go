package trigger

import (
	"encoding/json"
	"net/http"
	"os"

	"fmt"

	"io/ioutil"

	"github.com/golang/glog"
)

const (
	ApplicationJSON = "application/json"
)

type Trigger struct {
	TriggerConfig string
}

type Args struct {
	ShortCommitid string
	Commitid      string
	Branch        string
	TimeString    string
}

type EventInfo struct {
	PushData   PushData   `json:"push_data"`
	Repository Repository `json:"repository"`
}
type PushData struct {
	Tag string `json:"tag"`
}
type Repository struct {
	RepoFullName string `json:"repo_full_name"`
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

func (dp *Trigger) Event(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Printf("%s", body)
	ei := &EventInfo{}
	json.Unmarshal(body, ei)
	dp.pullRequestMergedEvent(ei)

}

const t = `{"push_data":{"digest":"sha256:5c15432f9284d16c71e05ac271ac135bfc32c0905f23bf644e69a2da583901b9","pushed_at":"2019-12-02 17:53:03","tag":"v2_6aad4833-20191202175300"},"repository":{"date_created":"2019-06-15 16:09:27","name":"deployer-trigger","namespace":"knative-sample","region":"cn-hangzhou","repo_authentication_type":"NO_CERTIFIED","repo_full_name":"knative-sample/deployer-trigger","repo_origin_type":"NO_CERTIFIED","repo_type":"PUBLIC"}}`
