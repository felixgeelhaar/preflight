package fleet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestHost(t *testing.T, id, hostname string) *Host {
	t.Helper()
	hostID, err := NewHostID(id)
	require.NoError(t, err)
	host, err := NewHost(hostID, SSHConfig{Hostname: hostname})
	require.NoError(t, err)
	return host
}

func TestNewInventory(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	assert.NotNil(t, inv)
	assert.Equal(t, 0, inv.HostCount())
	assert.Equal(t, 0, inv.GroupCount())
	assert.Equal(t, 22, inv.Defaults().Port)
}

func TestInventory_SetDefaults(t *testing.T) {
	t.Parallel()

	inv := NewInventory()
	inv.SetDefaults(SSHConfig{
		Port:           2222,
		User:           "admin",
		ConnectTimeout: 60 * time.Second,
	})

	defaults := inv.Defaults()
	assert.Equal(t, 2222, defaults.Port)
	assert.Equal(t, "admin", defaults.User)
	assert.Equal(t, 60*time.Second, defaults.ConnectTimeout)
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestInventory_Hosts(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	t.Run("add host", func(t *testing.T) {
		host := createTestHost(t, "server01", "10.0.0.1")
		err := inv.AddHost(host)
		require.NoError(t, err)
		assert.Equal(t, 1, inv.HostCount())
	})

	t.Run("add nil host", func(t *testing.T) {
		err := inv.AddHost(nil)
		assert.Error(t, err)
	})

	t.Run("add duplicate host", func(t *testing.T) {
		host := createTestHost(t, "server01", "10.0.0.2")
		err := inv.AddHost(host)
		assert.Error(t, err)
	})

	t.Run("get host", func(t *testing.T) {
		host, ok := inv.GetHost(HostID("server01"))
		assert.True(t, ok)
		assert.Equal(t, "10.0.0.1", host.SSH().Hostname)
	})

	t.Run("get nonexistent host", func(t *testing.T) {
		_, ok := inv.GetHost(HostID("nonexistent"))
		assert.False(t, ok)
	})

	t.Run("all hosts", func(t *testing.T) {
		host2 := createTestHost(t, "server02", "10.0.0.2")
		_ = inv.AddHost(host2)

		hosts := inv.AllHosts()
		assert.Len(t, hosts, 2)
	})

	t.Run("remove host", func(t *testing.T) {
		removed := inv.RemoveHost(HostID("server02"))
		assert.True(t, removed)
		assert.Equal(t, 1, inv.HostCount())
	})

	t.Run("remove nonexistent host", func(t *testing.T) {
		removed := inv.RemoveHost(HostID("nonexistent"))
		assert.False(t, removed)
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestInventory_Groups(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	t.Run("add group", func(t *testing.T) {
		name, _ := NewGroupName("production")
		group := NewGroup(name)
		err := inv.AddGroup(group)
		require.NoError(t, err)
		assert.Equal(t, 1, inv.GroupCount())
	})

	t.Run("add nil group", func(t *testing.T) {
		err := inv.AddGroup(nil)
		assert.Error(t, err)
	})

	t.Run("add duplicate group", func(t *testing.T) {
		name, _ := NewGroupName("production")
		group := NewGroup(name)
		err := inv.AddGroup(group)
		assert.Error(t, err)
	})

	t.Run("get group", func(t *testing.T) {
		group, ok := inv.GetGroup(GroupName("production"))
		assert.True(t, ok)
		assert.Equal(t, GroupName("production"), group.Name())
	})

	t.Run("get nonexistent group", func(t *testing.T) {
		_, ok := inv.GetGroup(GroupName("nonexistent"))
		assert.False(t, ok)
	})

	t.Run("all groups", func(t *testing.T) {
		name2, _ := NewGroupName("staging")
		group2 := NewGroup(name2)
		_ = inv.AddGroup(group2)

		groups := inv.AllGroups()
		assert.Len(t, groups, 2)
	})

	t.Run("remove group", func(t *testing.T) {
		removed := inv.RemoveGroup(GroupName("staging"))
		assert.True(t, removed)
		assert.Equal(t, 1, inv.GroupCount())
	})
}

func TestInventory_HostsByTag(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	host1 := createTestHost(t, "server01", "10.0.0.1")
	host1.AddTag(MustTag("darwin"))
	host1.AddTag(MustTag("production"))
	_ = inv.AddHost(host1)

	host2 := createTestHost(t, "server02", "10.0.0.2")
	host2.AddTag(MustTag("linux"))
	host2.AddTag(MustTag("production"))
	_ = inv.AddHost(host2)

	host3 := createTestHost(t, "server03", "10.0.0.3")
	host3.AddTag(MustTag("linux"))
	host3.AddTag(MustTag("staging"))
	_ = inv.AddHost(host3)

	t.Run("by single tag", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByTag(MustTag("darwin"))
		assert.Len(t, hosts, 1)
		assert.Equal(t, HostID("server01"), hosts[0].ID())
	})

	t.Run("by tag with multiple matches", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByTag(MustTag("production"))
		assert.Len(t, hosts, 2)
	})

	t.Run("by nonexistent tag", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByTag(MustTag("windows"))
		assert.Empty(t, hosts)
	})
}

func TestInventory_HostsByTags(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	host1 := createTestHost(t, "server01", "10.0.0.1")
	host1.AddTag(MustTag("darwin"))
	_ = inv.AddHost(host1)

	host2 := createTestHost(t, "server02", "10.0.0.2")
	host2.AddTag(MustTag("linux"))
	_ = inv.AddHost(host2)

	tags, _ := NewTags("darwin", "windows")
	hosts := inv.HostsByTags(tags)
	assert.Len(t, hosts, 1)
	assert.Equal(t, HostID("server01"), hosts[0].ID())
}

func TestInventory_HostsByGroup(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	// Add hosts
	host1 := createTestHost(t, "web-01", "10.0.0.1")
	host1.AddGroup("web")
	_ = inv.AddHost(host1)

	host2 := createTestHost(t, "web-02", "10.0.0.2")
	_ = inv.AddHost(host2)

	host3 := createTestHost(t, "db-01", "10.0.0.3")
	_ = inv.AddHost(host3)

	// Add group with pattern
	name, _ := NewGroupName("web")
	group := NewGroup(name)
	group.AddHostPattern("web-*")
	_ = inv.AddGroup(group)

	t.Run("by direct membership and pattern", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByGroup(GroupName("web"))
		assert.Len(t, hosts, 2) // Both web-01 (direct) and web-02 (pattern)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByGroup(GroupName("nonexistent"))
		assert.Empty(t, hosts)
	})
}

func TestInventory_HostsByPattern(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	_ = inv.AddHost(createTestHost(t, "web-01", "10.0.0.1"))
	_ = inv.AddHost(createTestHost(t, "web-02", "10.0.0.2"))
	_ = inv.AddHost(createTestHost(t, "db-01", "10.0.0.3"))
	_ = inv.AddHost(createTestHost(t, "db-02", "10.0.0.4"))

	t.Run("wildcard pattern", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByPattern("web-*")
		assert.Len(t, hosts, 2)
	})

	t.Run("single char wildcard", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByPattern("db-0?")
		assert.Len(t, hosts, 2)
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		hosts := inv.HostsByPattern("cache-*")
		assert.Empty(t, hosts)
	})
}

func TestInventory_HostsByStatus(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	host1 := createTestHost(t, "server01", "10.0.0.1")
	host1.MarkOnline()
	_ = inv.AddHost(host1)

	host2 := createTestHost(t, "server02", "10.0.0.2")
	host2.MarkOffline()
	_ = inv.AddHost(host2)

	host3 := createTestHost(t, "server03", "10.0.0.3")
	// Remains unknown
	_ = inv.AddHost(host3)

	assert.Len(t, inv.HostsByStatus(HostStatusOnline), 1)
	assert.Len(t, inv.HostsByStatus(HostStatusOffline), 1)
	assert.Len(t, inv.HostsByStatus(HostStatusUnknown), 1)
}

func TestInventory_ResolveGroupPolicies(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	// Create base group
	baseName, _ := NewGroupName("base")
	base := NewGroup(baseName)
	base.AddPolicy("notify-slack")
	_ = inv.AddGroup(base)

	// Create production group inheriting from base
	prodName, _ := NewGroupName("production")
	prod := NewGroup(prodName)
	prod.AddInherit(baseName)
	prod.AddPolicy("require-approval")
	_ = inv.AddGroup(prod)

	// Create critical group inheriting from production
	critName, _ := NewGroupName("critical")
	crit := NewGroup(critName)
	crit.AddInherit(prodName)
	crit.AddPolicy("maintenance-window")
	_ = inv.AddGroup(crit)

	t.Run("base policies", func(t *testing.T) {
		t.Parallel()
		policies := inv.ResolveGroupPolicies(baseName)
		assert.Contains(t, policies, "notify-slack")
		assert.Len(t, policies, 1)
	})

	t.Run("inherited policies", func(t *testing.T) {
		t.Parallel()
		policies := inv.ResolveGroupPolicies(prodName)
		assert.Contains(t, policies, "notify-slack")
		assert.Contains(t, policies, "require-approval")
		assert.Len(t, policies, 2)
	})

	t.Run("deeply inherited policies", func(t *testing.T) {
		t.Parallel()
		policies := inv.ResolveGroupPolicies(critName)
		assert.Contains(t, policies, "notify-slack")
		assert.Contains(t, policies, "require-approval")
		assert.Contains(t, policies, "maintenance-window")
		assert.Len(t, policies, 3)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		t.Parallel()
		policies := inv.ResolveGroupPolicies(GroupName("nonexistent"))
		assert.Empty(t, policies)
	})
}

func TestInventory_Summary(t *testing.T) {
	t.Parallel()

	inv := NewInventory()

	host1 := createTestHost(t, "server01", "10.0.0.1")
	host1.AddTag(MustTag("darwin"))
	host1.MarkOnline()
	_ = inv.AddHost(host1)

	host2 := createTestHost(t, "server02", "10.0.0.2")
	host2.AddTag(MustTag("linux"))
	host2.AddTag(MustTag("darwin"))
	host2.MarkOffline()
	_ = inv.AddHost(host2)

	name, _ := NewGroupName("production")
	_ = inv.AddGroup(NewGroup(name))

	summary := inv.Summary()

	assert.Equal(t, 2, summary.HostCount)
	assert.Equal(t, 1, summary.GroupCount)
	assert.Equal(t, 1, summary.OnlineCount)
	assert.Equal(t, 1, summary.OfflineCount)
	assert.Equal(t, 0, summary.ErrorCount)
	assert.Equal(t, 2, summary.TagCounts["darwin"])
	assert.Equal(t, 1, summary.TagCounts["linux"])
}
