import { describe, it, expect } from "vitest";
import { WorktreeFuzzySearch } from "./fuzzy.js";
import type { Worktree } from "../commands/list.js";

const mockWorktrees: Worktree[] = [
	{
		name: "main",
		branch: "main",
		path: "/repo/main",
		head: "abc123",
		active: true,
		locked: false,
	},
	{
		name: "feature-auth",
		branch: "feature/auth",
		path: "/repo/feature-auth",
		head: "def456",
		active: false,
		locked: false,
	},
	{
		name: "bugfix-login",
		branch: "bugfix/login",
		path: "/repo/bugfix-login",
		head: "ghi789",
		active: false,
		locked: false,
	},
];

describe("WorktreeFuzzySearch", () => {
	it("should return all worktrees when query is empty", () => {
		const fuzzy = new WorktreeFuzzySearch(mockWorktrees);
		const result = fuzzy.search("");
		expect(result).toEqual(mockWorktrees);
	});

	it("should filter worktrees based on name", () => {
		const fuzzy = new WorktreeFuzzySearch(mockWorktrees);
		const result = fuzzy.search("auth");
		expect(result).toHaveLength(1);
		expect(result[0]?.name).toBe("feature-auth");
	});

	it("should filter worktrees based on branch", () => {
		const fuzzy = new WorktreeFuzzySearch(mockWorktrees);
		const result = fuzzy.search("bugfix");
		expect(result).toHaveLength(1);
		expect(result[0]?.branch).toBe("bugfix/login");
	});

	it("should update worktrees collection", () => {
		const fuzzy = new WorktreeFuzzySearch(mockWorktrees);
		const newWorktrees: Worktree[] = [
			{
				name: "develop",
				branch: "develop",
				path: "/repo/develop",
				head: "xyz789",
				active: false,
				locked: false,
			},
		];
		fuzzy.update(newWorktrees);
		const result = fuzzy.search("");
		expect(result).toEqual(newWorktrees);
	});
});
