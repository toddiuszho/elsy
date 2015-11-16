package command

import (
  "os"
  "fmt"

  "github.com/codegangsta/cli"
  "github.com/fsouza/go-dockerclient"
  "github.com/Sirupsen/logrus"
  "stash0.eng.lancope.local/dev-infrastructure/project-lifecycle/helpers"
)

func CmdTeardown(c *cli.Context) error {
  if err := helpers.RunCommand(dockerComposeCommand(c, "rm", "-f", "-v")); err != nil {
    return err
  }

  // only remove containers that don't have the com.lancope.docker-gc.keep set
  client := helpers.GetDockerClient()
  project := fmt.Sprintf("com.docker.compose.project=%s", os.Getenv("COMPOSE_PROJECT_NAME"))
  queryAll := docker.ListContainersOptions{All: true, Filters: map[string][]string{"label": []string{project}}}
  queryGc := docker.ListContainersOptions{All: true,
    Filters: map[string][]string{"label": []string{project, "com.lancope.docker-gc.keep=True"}}}

  containers, err := client.ListContainers(queryAll)
  if err != nil {
    logrus.Errorf("could not query containers to remove", err)
    return err
  }
  logrus.Debugf("found %d container(s) for possible removal", len(containers))

  gcSafeContainers, err := client.ListContainers(queryGc)
  if err != nil {
    logrus.Errorf("could not query containers to remove", err)
    return err
  }
  logrus.Debugf("found %d container(s) with gc protection", len(gcSafeContainers))

  allIds := getContainerIds(&containers)
  gcSafeIds := getContainerIds(&gcSafeContainers)

  idsToRemove := removeIds(&allIds, &gcSafeIds)
  logrus.Debugf("removing %d containers", len(idsToRemove))
  for _, id := range idsToRemove {
    if err := client.RemoveContainer(docker.RemoveContainerOptions{ID: id, RemoveVolumes: true}); err != nil {
      logrus.Errorf("error removing container with ID: %s", id, err)
    }
  }
  return nil
}

func getContainerIds(contaners *[]docker.APIContainers) []string{
  ids := []string{}
  for _, container := range *contaners {
    ids = append(ids, container.ID)
  }
  return ids
}

// remove the items in the 2nd argument from the first
func removeIds(allIds *[]string, idsToRemove *[]string) []string {
  if len(*idsToRemove) <= 0 {
    return *allIds
  }

  ids := []string{}
  for _, id := range *allIds {
    keep := true
    for _, r := range *idsToRemove {
      if id == r {
        keep = false
      }
    }
    if keep {
      ids = append(ids, id)
    }
  }
  return ids
}
