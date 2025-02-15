/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package composer

import (
	"context"

	compose "github.com/compose-spec/compose-go/types"
	"github.com/containerd/nerdctl/pkg/composer/serviceparser"
	"github.com/containerd/nerdctl/pkg/reflectutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type DownOptions struct {
	RemoveVolumes bool
}

func (c *Composer) Down(ctx context.Context, downOptions DownOptions) error {
	if unknown := reflectutil.UnknownNonEmptyFields(c.project, "Name", "WorkingDir", "Services", "Networks", "Volumes", "ComposeFiles"); len(unknown) > 0 {
		logrus.Warnf("Ignoring: %+v", unknown)
	}

	for _, svc := range c.project.Services {
		if err := c.downService(ctx, svc, downOptions.RemoveVolumes); err != nil {
			return err
		}
	}

	for shortName := range c.project.Networks {
		if err := c.downNetwork(ctx, shortName); err != nil {
			return err
		}
	}

	if downOptions.RemoveVolumes {
		for shortName := range c.project.Volumes {
			if err := c.downVolume(ctx, shortName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Composer) downNetwork(ctx context.Context, shortName string) error {
	net, ok := c.project.Networks[shortName]
	if !ok {
		return errors.Errorf("invalid network name %q", shortName)
	}
	if net.External.External {
		// NOP
		return nil
	}
	// shortName is like "default", fullName is like "compose-wordpress_default"
	fullName := net.Name
	netExists, err := c.NetworkExists(fullName)
	if err != nil {
		return err
	} else if netExists {
		logrus.Infof("Removing network %s", fullName)
		if err := c.runNerdctlCmd(ctx, "network", "rm", fullName); err != nil {
			logrus.Warn(err)
		}
	}
	return nil
}

func (c *Composer) downVolume(ctx context.Context, shortName string) error {
	vol, ok := c.project.Volumes[shortName]
	if !ok {
		return errors.Errorf("invalid volume name %q", shortName)
	}
	if vol.External.External {
		// NOP
		return nil
	}
	// shortName is like "db_data", fullName is like "compose-wordpress_db_data"
	fullName := vol.Name
	volExists, err := c.VolumeExists(fullName)
	if err != nil {
		return err
	} else if volExists {
		logrus.Infof("Removing volume %s", fullName)
		if err := c.runNerdctlCmd(ctx, "volume", "rm", "-f", fullName); err != nil {
			logrus.Warn(err)
		}
	}
	return nil
}

func (c *Composer) downService(ctx context.Context, svc compose.ServiceConfig, removeAnonVolumes bool) error {
	ps, err := serviceparser.Parse(c.project, svc)
	if err != nil {
		return err
	}
	for _, container := range ps.Containers {
		logrus.Infof("Removing container %s", container.Name)
		args := []string{"rm", "-f"}
		if removeAnonVolumes {
			args = append(args, "-v")
		}
		args = append(args, container.Name)
		if err := c.runNerdctlCmd(ctx, args...); err != nil {
			logrus.Warn(err)
		}
	}
	return nil
}
