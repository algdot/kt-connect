package command

import (
	"context"
	"errors"
	"fmt"
	"github.com/alibaba/kt-connect/pkg/kt/command/general"
	"os"
	"strings"

	"github.com/alibaba/kt-connect/pkg/common"
	"github.com/alibaba/kt-connect/pkg/kt"
	"github.com/alibaba/kt-connect/pkg/kt/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/connect"
	"github.com/alibaba/kt-connect/pkg/kt/options"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/alibaba/kt-connect/pkg/process"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	urfave "github.com/urfave/cli"
	"k8s.io/api/apps/v1"
)

// NewMeshCommand return new mesh command
func NewMeshCommand(cli kt.CliInterface, options *options.DaemonOptions, action ActionInterface) urfave.Command {
	return urfave.Command{
		Name:  "mesh",
		Usage: "mesh kubernetes deployment to local",
		Flags: general.MeshActionFlag(options),
		Action: func(c *urfave.Context) error {
			if options.Debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			if err := general.CombineKubeOpts(options); err != nil {
				return err
			}
			resourceToMesh := c.Args().First()
			expose := options.MeshOptions.Expose

			if len(resourceToMesh) == 0 {
				return errors.New("name of deployment to mesh is required")
			}

			if len(expose) == 0 {
				return errors.New("--expose is required")
			}

			return action.Mesh(resourceToMesh, cli, options)
		},
	}
}

//Mesh exchange kubernetes workload
func (action *Action) Mesh(resourceName string, cli kt.CliInterface, options *options.DaemonOptions) error {
	ch, err := general.SetupProcess(cli, options, common.ComponentMesh)
	if err != nil {
		return err
	}

	if port := util.FindBrokenPort(options.MeshOptions.Expose); port != "" {
		return fmt.Errorf("no application is running on port %s", port)
	}

	kubernetes, err := cli.Kubernetes()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if options.MeshOptions.Method == common.MeshMethodManual {
		err = manualMesh(ctx, resourceName, kubernetes, options)
	} else if options.MeshOptions.Method == common.MeshMethodAuto {
		err = autoMesh(ctx, resourceName, kubernetes, options)
	} else {
		err = fmt.Errorf("invalid mesh method \"%s\"", options.MeshOptions.Method)
	}
	if err != nil {
		return err
	}

	// watch background process, clean the workspace and exit if background process occur exception
	go func() {
		<-process.Interrupt()
		log.Error().Msgf("Command interrupted", <-process.Interrupt())
		general.CleanupWorkspace(cli, options)
		os.Exit(0)
	}()

	s := <-ch
	log.Info().Msgf("Terminal Signal is %s", s)

	return nil
}

func manualMesh(ctx context.Context, deploymentName string, kubernetes cluster.KubernetesInterface, options *options.DaemonOptions) error {
	app, err := kubernetes.Deployment(ctx, deploymentName, options.Namespace)
	if err != nil {
		return err
	}

	meshVersion := getVersion(options)

	shadowPodName := app.GetObjectMeta().GetName() + "-kt-" + meshVersion
	labels := getMeshLabels(shadowPodName, meshVersion, app, options)

	if err = createShadowAndInbound(ctx, shadowPodName, labels, options, kubernetes); err != nil {
		return err
	}

	log.Info().Msg("---------------------------------------------------------")
	log.Info().Msgf("    Mesh Version '%s' You can update Istio rule       ", meshVersion)
	log.Info().Msg("---------------------------------------------------------")
	return nil
}

func autoMesh(ctx context.Context, deploymentName string, kubernetes cluster.KubernetesInterface, options *options.DaemonOptions) error {

	return nil
}

func createShadowAndInbound(ctx context.Context, shadowPodName string, labels map[string]string, options *options.DaemonOptions,
	kubernetes cluster.KubernetesInterface) error {

	envs := make(map[string]string)
	annotations := make(map[string]string)
	_, podName, sshConfigMapName, _, err := kubernetes.GetOrCreateShadow(ctx, shadowPodName, options, labels, annotations, envs)
	if err != nil {
		return err
	}

	// record context data
	options.RuntimeOptions.Shadow = shadowPodName
	options.RuntimeOptions.SSHCM = sshConfigMapName

	shadow := connect.Create(options)
	err = shadow.Inbound(options.MeshOptions.Expose, podName)

	if err != nil {
		return err
	}
	return nil
}

func getMeshLabels(workload string, meshVersion string, app *v1.Deployment, options *options.DaemonOptions) map[string]string {
	labels := map[string]string{
		common.ControlBy:   common.KubernetesTool,
		common.KTComponent: common.ComponentMesh,
		common.KTName:      workload,
		common.KTVersion:   meshVersion,
	}
	if app != nil {
		for k, v := range app.Spec.Selector.MatchLabels {
			labels[k] = v
		}
	}
	return labels
}

func getVersion(options *options.DaemonOptions) string {
	if len(options.MeshOptions.Version) != 0 {
		return options.MeshOptions.Version
	}
	return strings.ToLower(util.RandomString(5))
}
