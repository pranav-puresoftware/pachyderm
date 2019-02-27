package server

import (
	"fmt"
	"sync"

	"github.com/pachyderm/pachyderm/src/client"
	ppsclient "github.com/pachyderm/pachyderm/src/client/pps"
	"github.com/pachyderm/pachyderm/src/server/pkg/log"
	"github.com/pachyderm/pachyderm/src/server/pkg/metrics"
	"github.com/pachyderm/pachyderm/src/server/pkg/ppsconsts"
	"github.com/pachyderm/pachyderm/src/server/pkg/ppsdb"
	"github.com/pachyderm/pachyderm/src/server/pkg/serviceenv"
)

var initSpecRepoOnce sync.Once

func initSpecRepo(env *serviceenv.ServiceEnv, a *apiServer) {
	initSpecRepoOnce.Do(func() {
		// Initialize spec repo
		if err := a.sudo(
			env.GetPachClient(nil),
			func(superUserClient *client.APIClient) error {
				if err := superUserClient.CreateRepo(ppsconsts.SpecRepo); err != nil {
					if !isAlreadyExistsErr(err) {
						return err
					}
				}
				return nil
			}); err != nil {
			panic(fmt.Sprintf("could not create pipeline spec repo: %v", err))
		}
	})
}

// NewAPIServer creates an APIServer.
func NewAPIServer(
	env *serviceenv.ServiceEnv,
	etcdPrefix string,
	namespace string,
	workerImage string,
	workerSidecarImage string,
	workerImagePullPolicy string,
	storageRoot string,
	storageBackend string,
	storageHostPath string,
	iamRole string,
	imagePullSecret string,
	noExposeDockerSocket bool,
	reporter *metrics.Reporter,
	workerUsesRoot bool,
	workerGrpcPort uint16,
	port uint16,
	pprofPort uint16,
	httpPort uint16,
	peerPort uint16,
) (ppsclient.APIServer, error) {
	apiServer := &apiServer{
		Logger:                log.NewLogger("pps.API"),
		env:                   env,
		etcdPrefix:            etcdPrefix,
		namespace:             namespace,
		workerImage:           workerImage,
		workerSidecarImage:    workerSidecarImage,
		workerImagePullPolicy: workerImagePullPolicy,
		storageRoot:           storageRoot,
		storageBackend:        storageBackend,
		storageHostPath:       storageHostPath,
		iamRole:               iamRole,
		imagePullSecret:       imagePullSecret,
		noExposeDockerSocket:  noExposeDockerSocket,
		reporter:              reporter,
		workerUsesRoot:        workerUsesRoot,
		pipelines:             ppsdb.Pipelines(env.GetEtcdClient(), etcdPrefix),
		jobs:                  ppsdb.Jobs(env.GetEtcdClient(), etcdPrefix),
		monitorCancels:        make(map[string]func()),
		workerGrpcPort:        workerGrpcPort,
		port:                  port,
		pprofPort:             pprofPort,
		httpPort:              httpPort,
		peerPort:              peerPort,
	}
	apiServer.validateKube()
	go initSpecRepo(env, apiServer)
	return apiServer, nil
}

// NewSidecarAPIServer creates an APIServer that has limited functionalities
// and is meant to be run as a worker sidecar.  It cannot, for instance,
// create pipelines.
func NewSidecarAPIServer(
	env *serviceenv.ServiceEnv,
	etcdPrefix string,
	iamRole string,
	reporter *metrics.Reporter,
	workerGrpcPort uint16,
	pprofPort uint16,
	httpPort uint16,
	peerPort uint16,
) (ppsclient.APIServer, error) {
	apiServer := &apiServer{
		Logger:         log.NewLogger("pps.API"),
		env:            env,
		etcdPrefix:     etcdPrefix,
		iamRole:        iamRole,
		reporter:       reporter,
		workerUsesRoot: true,
		pipelines:      ppsdb.Pipelines(env.GetEtcdClient(), etcdPrefix),
		jobs:           ppsdb.Jobs(env.GetEtcdClient(), etcdPrefix),
		workerGrpcPort: workerGrpcPort,
		pprofPort:      pprofPort,
		httpPort:       httpPort,
		peerPort:       peerPort,
	}
	go initSpecRepo(env, apiServer)
	return apiServer, nil
}
