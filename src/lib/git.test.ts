import { describe, it, expect } from "vitest";
import { parseWorktreeList, GitOperationError } from "./git.js";
import type { Worktree } from "../commands/list.js";

describe("Git operations", () => {
	describe("parseWorktreeList", () => {
		it("should parse worktree list output correctly", () => {
			const output = `worktree /repo/main
HEAD abc123f456
branch refs/heads/main

worktree /repo/feature-auth
HEAD def456a789
branch refs/heads/feature/auth

worktree /repo/bugfix-login
HEAD ghi789b012
branch refs/heads/bugfix/login
locked`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(3);

			expect(result[0]).toEqual({
				name: "main",
				path: "/repo/main",
				head: "abc123f456",
				branch: "main",
				active: false,
				locked: false,
			});

			expect(result[1]).toEqual({
				name: "feature-auth",
				path: "/repo/feature-auth",
				head: "def456a789",
				branch: "feature/auth",
				active: false,
				locked: false,
			});

			expect(result[2]).toEqual({
				name: "bugfix-login",
				path: "/repo/bugfix-login",
				head: "ghi789b012",
				branch: "bugfix/login",
				active: false,
				locked: true,
			});
		});

		it("should handle detached HEAD worktrees", () => {
			const output = `worktree /repo/detached
HEAD abc123f456
detached`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(1);
			expect(result[0]?.branch).toBe("HEAD");
		});

		it("should handle empty output", () => {
			const result = parseWorktreeList("");
			expect(result).toHaveLength(0);
		});
	});

	describe("GitOperationError", () => {
		it("should create error with code and stderr", () => {
			const error = new GitOperationError(
				"Git command failed",
				"GIT_ERROR",
				"stderr output",
			);

			expect(error.message).toBe("Git command failed");
			expect(error.code).toBe("GIT_ERROR");
			expect(error.stderr).toBe("stderr output");
			expect(error.name).toBe("GitOperationError");
		});
	});
});
