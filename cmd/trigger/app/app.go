package app

import (
	"strings"

	"os"

	"time"

	"github.com/golang/glog"
	"github.com/knative-sample/deployer/cmd/trigger/app/options"
	"github.com/knative-sample/deployer/pkg/trigger"
	"github.com/spf13/cobra"
)

func NewCommandStartServer(stopCh <-chan struct{}) *cobra.Command {
	ops := &options.Options{}
	mainCmd := &cobra.Command{
		Short: "Knative github trigger ",
		Long:  "Knative github trigger ",
		RunE: func(c *cobra.Command, args []string) error {
			glog.V(2).Infof("NewCommandStartServer main:%s", strings.Join(args, " "))
			run(stopCh, ops)
			return nil
		},
	}

	ops.SetOps(mainCmd)
	return mainCmd
}

// run command
func run(stopCh <-chan struct{}, ops *options.Options) {
	if ops.TriggerConfig == "" {
		glog.Fatalf("--trigger-config is empty")
	}

	tg := trigger.Trigger{
		TriggerConfig: ops.TriggerConfig,
	}

	go func() {
		<-stopCh
		time.Sleep(time.Second)
		os.Exit(0)
	}()

	if err := tg.Run(); err != nil {
		glog.Errorf("dp.Run error:%s", err)
	}
}
