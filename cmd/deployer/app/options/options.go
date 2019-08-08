package options

import (
	"github.com/spf13/cobra"
)

type Options struct {
	Image       string
	Namespace   string
	ServiceName string
	Port        string
}

func (s *Options) SetOps(ac *cobra.Command) {
	ac.Flags().StringVar(&s.Image, "image", s.Image, "image")
	ac.Flags().StringVar(&s.Namespace, "namespace", "default", "namespace")
	ac.Flags().StringVar(&s.ServiceName, "serivce-name", s.ServiceName, "Knative service name")
	ac.Flags().StringVar(&s.Port, "port", "8080", "port")
}
