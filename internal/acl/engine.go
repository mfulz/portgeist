// Package acl provides a simple role- and group-based access control layer
// that supports both global and object-specific permission checks.
// It is designed for easy integration with YAML-configured CLI applications
// and allows defining users, groups, roles, and rule-based ACLs.
//
// Example usage:
//
//	acl.Init(cfg)
//	if !acl.Can("userx", "proxy_start") {
//		return errors.New("permission denied")
//	}
package acl

import (
	"fmt"
	"slices"

	"github.com/mfulz/portgeist/internal/logging"
	"github.com/mfulz/portgeist/protocol"
)

// Permission defines a named right or capability.
type Permission string

// ACLRuleSet defines a set of permissions for a proxy or other object.
type ACLRuleSet struct {
	// TODO defaults?
	Rules []ACLRule `mapstructure:"rules"`
}

// ACLRule defines permissions for a proxy or other object.
type ACLRule struct {
	Description string       `mapstructure:"description"`
	Subjects    []string     `mapstructure:"subjects"`
	Permissions []Permission `mapstructure:"permissions,omitempty"`
	Deny        bool         `mapstructure:"deny"`
}

// User defines a named user (e.g. login name).
type User struct {
	Name   string   `mapstructure:"name"`
	Roles  []string `mapstructure:"roles"`
	Token  string   `mapstructure:"token"`
	groups []string
}

// Group defines a named group of users.
type Group struct {
	Name    string   `mapstructure:"name"`
	Members []string `mapstructure:"members"`
	Roles   []string `mapstructure:"roles"`
}

// Role defines a named role, grouping one or more permissions.
type Role struct {
	Name        string       `mapstructure:"name"`
	Permissions []Permission `mapstructure:"permissions"`
}

// ACLConfig defines the global ACL structure loaded from config.
type ACLConfig struct {
	Enabled bool             `mapstructure:"enabled"`
	Users   map[string]User  `mapstructure:"users"`
	Groups  map[string]Group `mapstructure:"groups"`
	Roles   map[string]Role  `mapstructure:"roles"`
}

// aclChecker represents the internal ACL state and evaluation logic.
type aclChecker struct {
	enabled bool
	users   map[string]User
	groups  map[string]Group
	roles   map[string]Role
}

// aclhandle is the globally accessible instance used for all ACL checks.
var aclhandle *aclChecker

// validPerms holds the accepted set of permission names passed at Init time.
var validPerms map[Permission]struct{}

// Init initializes the global ACL engine from config.
func Init(cfg ACLConfig, perms []Permission) error {
	pmap := make(map[Permission]struct{}, len(perms))
	for _, p := range perms {
		pmap[p] = struct{}{}
	}

	// Validate roles
	for roleName, role := range cfg.Roles {
		for _, perm := range role.Permissions {
			if _, ok := pmap[perm]; !ok {
				return fmt.Errorf("invalid permission '%s' in role '%s'", perm, roleName)
			}
		}
		if role.Name == "" {
			role.Name = roleName
			cfg.Roles[roleName] = role
		}
	}

	// Validate users
	for name, user := range cfg.Users {
		if user.Name == "" {
			user.Name = name
			cfg.Users[name] = user
		}
	}

	// Validate groups
	for name, group := range cfg.Groups {
		if group.Name == "" {
			group.Name = name
			cfg.Groups[name] = group
		}

		for _, member := range group.Members {
			if u, ok := cfg.Users[member]; ok {
				u.groups = append(u.groups, group.Name)
				cfg.Users[member] = u
				continue
			}
			// if user not existing error out
			return fmt.Errorf("invalid user '%s' in group '%s'", member, group.Name)
		}
	}

	aclhandle = &aclChecker{
		enabled: cfg.Enabled,
		users:   cfg.Users,
		groups:  cfg.Groups,
		roles:   cfg.Roles,
	}

	validPerms = pmap
	return nil
}

// aclValid checks whether ACLs are ready and enabled.
// Returns (true, false) → reject: uninitialized
// Returns (true, true)  → allow: disabled in config
// Returns (false, _)    → continue with normal check
func aclValid() (bool, bool) {
	if aclhandle == nil {
		return true, false
	}
	if !aclhandle.enabled {
		return true, true
	}
	return false, false
}

// Can checks whether the given user has the specified global permission.
func Can(user string, perm Permission, rules ACLRuleSet) bool {
	if handled, result := aclValid(); handled {
		return result
	}
	return aclhandle.can(user, perm, rules)
}

// hasPerm checks if the ACLRule has the permission. If perms are empty it matches all
func (r *ACLRule) hasPerm(perm Permission) bool {
	if len(r.Permissions) == 0 {
		return true
	}

	return slices.Contains(r.Permissions, perm)
}

// hasSubject checks if the ACLRule has the subject.
func (r *ACLRule) hasSubject(user string) bool {
	if len(r.Subjects) == 0 {
		return false
	}

	for _, s := range r.Subjects {
		if aclhandle.userMatches(user, s) {
			return true
		}
	}

	return false
}

// Authenticate checks if user uses correct token
func Authenticate(authReq *protocol.Auth) bool {
	if handled, result := aclValid(); handled {
		return result
	}

	if authReq == nil {
		return false
	}

	return aclhandle.userCredsValid(authReq.User, authReq.Token)
}

// authenticate authenticate the user by token verification
func (a *aclChecker) userCredsValid(user, token string) bool {
	u, ok := a.users[user]
	if !ok {
		return false
	}

	return u.Token == token
}

// matchRules checks if the actual user is matching the acl rules
func (a *aclChecker) can(user string, perm Permission, rules ACLRuleSet) bool {
	matches := false

	logging.Log.Debugf("Ruleset: %v", rules)

	if !a.userHasPermission(user, perm) {
		return false
	}

	if len(rules.Rules) == 0 {
		// all roles, groups are allowed just permission needs to be checked
		return true
	}

	for _, rule := range rules.Rules {
		if !rule.hasPerm(perm) {
			continue
		}

		if !rule.hasSubject(user) {
			continue
		}

		if rule.Deny {
			return false
		}
		matches = true
	}

	return matches
}

// userHasPermission checks if user belongs to group that owns a role.
func (a *aclChecker) userRoles(user User) []string {
	var ret []string

	ret = append(ret, user.Roles...)

	for _, groupName := range user.groups {
		if group, ok := a.groups[groupName]; ok {
			ret = append(ret, group.Roles...)
		}
	}

	return ret
}

// userHasPermission checks if user belongs to group that owns a role.
func (a *aclChecker) userHasPermission(user string, perm Permission) bool {
	u, ok := a.users[user]
	if !ok {
		return false
	}

	uroles := a.userRoles(u)
	for _, roleName := range uroles {
		if role, ok := a.roles[roleName]; ok {
			if slices.Contains(role.Permissions, perm) {
				return true
			}
		}
	}

	return false
}

// userMatches returns true if the subject matches the user or one of their groups.
func (a *aclChecker) userMatches(user string, subject string) bool {
	if subject == user {
		return true
	}
	u, ok := a.users[user]
	if !ok {
		return false
	}
	return slices.Contains(u.groups, subject)
}
