package deploy

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/component"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// All deploys all deployments in the config
func All(config *latest.Config, cache *generated.CacheConfig, client kubernetes.Interface, isDev, forceDeploy bool, builtImages map[string]string, log log.Logger) error {
	if config.Deployments != nil {
		for _, deployConfig := range *config.Deployments {
			var deployClient deploy.Interface
			var err error
			var method string

			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(config, client, deployConfig, log)
				if err != nil {
					return fmt.Errorf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}

				method = "kubectl"
			} else if deployConfig.Helm != nil {
				deployClient, err = helm.New(config, client, deployConfig, log)
				if err != nil {
					return fmt.Errorf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}

				method = "helm"
			} else if deployConfig.Component != nil {
				deployClient, err = component.New(config, client, deployConfig, log)
				if err != nil {
					return fmt.Errorf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}

				method = "component"
			} else {
				return fmt.Errorf("Error deploying devspace: deployment %s has no deployment method", *deployConfig.Name)
			}

			wasDeployed, err := deployClient.Deploy(cache, forceDeploy, builtImages)
			if err != nil {
				return fmt.Errorf("Error deploying %s: %v", *deployConfig.Name, err)
			}

			if wasDeployed {
				log.Donef("Successfully deployed %s with %s", *deployConfig.Name, method)
			} else {
				log.Infof("Skipping deployment %s", *deployConfig.Name)
			}
		}
	}

	return nil
}

// PurgeDeployments removes all deployments or a set of deployments from the cluster
func PurgeDeployments(config *latest.Config, cache *generated.CacheConfig, client kubernetes.Interface, deployments []string) {
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	if config.Deployments != nil {
		// Reverse them
		for i := len(*config.Deployments) - 1; i >= 0; i-- {
			deployConfig := (*config.Deployments)[i]

			// Check if we should skip deleting deployment
			if deployments != nil {
				found := false

				for _, value := range deployments {
					if value == *deployConfig.Name {
						found = true
						break
					}
				}

				if found == false {
					continue
				}
			}

			var err error
			var deployClient deploy.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config: %v", err)
					continue
				}
			} else if deployConfig.Helm != nil {
				deployClient, err = helm.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config: %v", err)
					continue
				}
			} else if deployConfig.Component != nil {
				deployClient, err = component.New(config, client, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create component deploy config: %v", err)
					continue
				}
			}

			log.StartWait("Deleting deployment " + *deployConfig.Name)
			err = deployClient.Delete(cache)
			log.StopWait()
			if err != nil {
				log.Warnf("Error deleting deployment %s: %v", *deployConfig.Name, err)
			}

			log.Donef("Successfully deleted deployment %s", *deployConfig.Name)
		}
	}
}
