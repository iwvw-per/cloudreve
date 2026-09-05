package workflows

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

const (
	TaskTempPath                 = "fm_workflows"
	slaveProgressRefreshInterval = 5 * time.Second

	// GroupPermissionAllowSelectNode is the group permission bit for "allow select node",
	// matching the frontend GroupPermission.allow_select_node.
	GroupPermissionAllowSelectNode = types.GroupPermission(15)
)

type NodeState struct {
	NodeID int `json:"node_id"`

	progress queue.Progresses
}

// ResolveTaskNodeID determines which node a background task should be assigned to,
// based on the owning group's node-selection settings and the user's preferred node.
// An empty AllowedNodes means all active nodes are permitted; returning 0 delegates
// the weighted selection to the node pool.
func ResolveTaskNodeID(ctx context.Context, dep dependency.Dep, group *ent.Group, preferred int) (int, error) {
	var allowedNodes []int
	canSelect := false
	if group != nil {
		if group.Settings != nil {
			allowedNodes = group.Settings.AllowedNodes
		}
		if group.Permissions != nil {
			canSelect = group.Permissions.Enabled(int(GroupPermissionAllowSelectNode))
		}
	}

	// Groups not allowed to select nodes cannot override the system choice.
	if !canSelect {
		preferred = 0
	}

	nodes, err := dep.NodeClient().ListActiveNodes(ctx, allowedNodes)
	if err != nil {
		return 0, fmt.Errorf("failed to list allowed nodes: %w", err)
	}
	if len(nodes) == 0 {
		return 0, fmt.Errorf("no available node: %w", cluster.ErrNoAvailableNode)
	}

	if preferred > 0 {
		for _, n := range nodes {
			if n.ID == preferred {
				return preferred, nil
			}
		}
		return 0, serializer.NewError(serializer.CodeNoPermissionErr, "Node is not allowed for this group", nil)
	}

	// No explicit allow list: let the node pool pick by weight among all active nodes.
	if len(allowedNodes) == 0 {
		return 0, nil
	}

	// Restrict auto-selection to the group's allowed nodes, picking the highest weight one.
	selected := nodes[0]
	for _, n := range nodes[1:] {
		if n.Weight > selected.Weight {
			selected = n
		}
	}
	return selected.ID, nil
}


// allocateNode allocates a node for the task.
func allocateNode(ctx context.Context, dep dependency.Dep, state *NodeState, capability types.NodeCapability) (cluster.Node, error) {
	np, err := dep.NodePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node pool: %w", err)
	}

	node, err := np.Get(ctx, capability, state.NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	state.NodeID = node.ID()
	return node, nil
}

// prepareSlaveTaskCtx prepares the context for the slave task.
func prepareSlaveTaskCtx(ctx context.Context, props *types.SlaveTaskProps) context.Context {
	ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, strconv.Itoa(props.NodeID))
	ctx = context.WithValue(ctx, cluster.MasterSiteUrlCtx{}, props.MasterSiteURl)
	ctx = context.WithValue(ctx, cluster.MasterSiteVersionCtx{}, props.MasterSiteVersion)
	ctx = context.WithValue(ctx, cluster.MasterSiteIDCtx{}, props.MasterSiteID)
	return ctx
}

func prepareTempFolder(ctx context.Context, dep dependency.Dep, t queue.Task) (string, error) {
	settings := dep.SettingProvider()
	tempPath := util.DataPath(path.Join(settings.TempPath(ctx), TaskTempPath, strconv.Itoa(t.ID())))
	if err := util.CreatNestedFolder(tempPath); err != nil {
		return "", fmt.Errorf("failed to create temp folder: %w", err)
	}

	dep.Logger().Info("Temp folder created: %s", tempPath)
	return tempPath, nil
}
