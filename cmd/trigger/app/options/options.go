package options

import (
	"github.com/spf13/cobra"
)

type Options struct {
	TriggerConfig string
}

func (s *Options) SetOps(ac *cobra.Command) {
	ac.Flags().StringVar(&s.TriggerConfig, "trigger-config", s.TriggerConfig, "trigger config")
}
