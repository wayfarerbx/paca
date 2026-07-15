export interface PermissionDefinition {
	key: string;
	domain: string;
}

export type PermissionMap = Record<string, boolean>;

export function hasPermission(
	grantedPermissions: string[],
	requiredPermission: string,
): boolean {
	if (grantedPermissions.includes("*")) return true;
	if (grantedPermissions.includes(requiredPermission)) return true;

	const lastDotIndex = requiredPermission.lastIndexOf(".");
	if (lastDotIndex === -1) return false;

	const prefix = requiredPermission.slice(0, lastDotIndex);
	return grantedPermissions.includes(`${prefix}.*`);
}

export function hasAnyPermission(
	grantedPermissions: string[],
	requiredPermissions: string[],
): boolean {
	return requiredPermissions.some((permission) =>
		hasPermission(grantedPermissions, permission),
	);
}

export function expandWildcardPermissions(
	source: PermissionMap | undefined,
	knownPermissions: PermissionDefinition[],
): PermissionMap {
	if (!source) return {};

	const expanded: PermissionMap = {};
	const hasGlobalWildcard = source["*"] === true;

	for (const permission of knownPermissions) {
		const domainWildcard = `${permission.domain}.*`;
		expanded[permission.key] =
			hasGlobalWildcard ||
			source[domainWildcard] === true ||
			source[permission.key] === true;
	}

	return expanded;
}

export function normalizePermissionsToWildcards(
	source: PermissionMap,
	knownPermissions: PermissionDefinition[],
): PermissionMap {
	if (source["*"] === true) {
		return { "*": true };
	}

	// Group by each permission's own key prefix (the part before its last
	// dot), NOT by `domain`. `domain` is a UI-grouping label and, for
	// plugin-declared permissions, multiple unrelated plugins can share the
	// synthetic "plugins" domain — collapsing by that label would produce a
	// bogus "plugins.*" key that (a) doesn't match any real permission check
	// (hasPermission only understands `${realPrefix}.*`) and (b) would
	// over-grant every other plugin's permissions sharing that UI group.
	// Using the key's real prefix keeps wildcard-collapsing scoped to
	// permissions that actually share a checkable namespace.
	const permissionsByPrefix = new Map<string, PermissionDefinition[]>();
	for (const permission of knownPermissions) {
		const lastDotIndex = permission.key.lastIndexOf(".");
		const prefix =
			lastDotIndex === -1 ? permission.key : permission.key.slice(0, lastDotIndex);
		const existing = permissionsByPrefix.get(prefix) ?? [];
		existing.push(permission);
		permissionsByPrefix.set(prefix, existing);
	}

	const normalized: PermissionMap = {};
	for (const [prefix, prefixPermissions] of permissionsByPrefix) {
		const enabledPermissions = prefixPermissions.filter(
			(permission) => source[permission.key] === true,
		);
		if (enabledPermissions.length === 0) continue;

		if (enabledPermissions.length === prefixPermissions.length) {
			normalized[`${prefix}.*`] = true;
			continue;
		}

		for (const permission of enabledPermissions) {
			normalized[permission.key] = true;
		}
	}

	return normalized;
}
