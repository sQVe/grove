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

		it("should handle bare repository entries", () => {
			const output = `worktree /repo
HEAD abc123f456
bare`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(1);
			expect(result[0]?.name).toBe("repo");
		});

		it("should handle worktrees with special characters in paths", () => {
			const output = `worktree /repo/feature with spaces
HEAD abc123f456
branch refs/heads/feature-spaces

worktree /repo/feature-中文
HEAD def456a789
branch refs/heads/feature-chinese`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(2);
			expect(result[0]?.name).toBe("feature with spaces");
			expect(result[0]?.path).toBe("/repo/feature with spaces");
			expect(result[1]?.name).toBe("feature-中文");
			expect(result[1]?.path).toBe("/repo/feature-中文");
		});

		it("should handle worktrees without branch info", () => {
			const output = `worktree /repo/broken
HEAD abc123f456`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(1);
			expect(result[0]?.branch).toBeUndefined();
		});

		it("should handle multiple locked worktrees", () => {
			const output = `worktree /repo/main
HEAD abc123f456
branch refs/heads/main

worktree /repo/locked1
HEAD def456a789
branch refs/heads/feature1
locked

worktree /repo/locked2
HEAD ghi789b012
branch refs/heads/feature2
locked`;

			const result = parseWorktreeList(output);

			expect(result).toHaveLength(3);
			expect(result[0]?.locked).toBe(false);
			expect(result[1]?.locked).toBe(true);
			expect(result[2]?.locked).toBe(true);
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

		it("should create error without code and stderr", () => {
			const error = new GitOperationError("Simple error");

			expect(error.message).toBe("Simple error");
			expect(error.code).toBeUndefined();
			expect(error.stderr).toBeUndefined();
			expect(error.name).toBe("GitOperationError");
		});

		it("should be instanceof Error", () => {
			const error = new GitOperationError("Test error");
			expect(error).toBeInstanceOf(Error);
			expect(error).toBeInstanceOf(GitOperationError);
		});
	});
});
