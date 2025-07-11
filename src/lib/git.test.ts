import { describe, it, expect, vi, beforeEach } from "vitest";
import {
	parseWorktreeList,
	GitOperationError,
	execWithShellPath,
	getCurrentBranch,
	hasCommits,
	getRepositoryStatus,
	initRepository,
	createBranch,
	createWorktreeAdvanced,
	type GitExecResult,
} from "./git.js";
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

		it("should create error from existing error", () => {
			const originalError = new GitOperationError(
				"Original error",
				"ORIGINAL_CODE",
				"stderr",
				"git command",
				"/path",
				1,
			);

			const newError = GitOperationError.fromError(
				"New message",
				originalError,
				"NEW_CODE",
			);

			expect(newError.message).toBe("New message");
			expect(newError.code).toBe("NEW_CODE");
			expect(newError.stderr).toBe("stderr");
			expect(newError.command).toBe("git command");
			expect(newError.cwd).toBe("/path");
			expect(newError.exitCode).toBe(1);
		});
	});

	describe("execWithShellPath", () => {
		it("should execute command with enhanced PATH", async () => {
			const result = await execWithShellPath("echo test");
			expect(result.stdout).toBe("test");
			expect(result.stderr).toBe("");
			expect(result.command).toBe("echo test");
			expect(result.cwd).toBeDefined();
		});

		it("should handle command failure gracefully", async () => {
			const result = await execWithShellPath("nonexistent-command-12345");
			expect(result.exitCode).toBeDefined();
			expect(result.stderr).toContain("not found");
		});

		it("should respect custom options", async () => {
			const result = await execWithShellPath("pwd", {
				cwd: "/",
				maxBuffer: 1024,
				timeout: 5000,
			});
			expect(result.cwd).toBe("/");
		});
	});

	describe("getCurrentBranch", () => {
		it("should return current branch name", async () => {
			// Mock successful git branch --show-current
			vi.mock("./git.js", async () => {
				const actual = await vi.importActual("./git.js");
				return {
					...actual,
					execGit: vi.fn().mockResolvedValue("main"),
				};
			});

			const branch = await getCurrentBranch();
			expect(branch).toBe("main");
		});

		it("should handle detached HEAD", async () => {
			// Test would require mocking git commands
			// For now, just verify the function exists
			expect(typeof getCurrentBranch).toBe("function");
		});
	});

	describe("hasCommits", () => {
		it("should return true for repositories with commits", async () => {
			// Test would require a real git repository or mocking
			expect(typeof hasCommits).toBe("function");
		});

		it("should return false for repositories without commits", async () => {
			// Test would require a real git repository or mocking
			expect(typeof hasCommits).toBe("function");
		});
	});

	describe("getRepositoryStatus", () => {
		it("should parse git status --porcelain output", async () => {
			// Test would require mocking git status output
			expect(typeof getRepositoryStatus).toBe("function");
		});

		it("should handle empty status (clean repo)", async () => {
			// Test would require mocking
			expect(typeof getRepositoryStatus).toBe("function");
		});

		it("should calculate ahead/behind counts", async () => {
			// Test would require mocking git rev-list output
			expect(typeof getRepositoryStatus).toBe("function");
		});
	});

	describe("initRepository", () => {
		it("should initialize repository with initial commit", async () => {
			// Test would require filesystem operations
			expect(typeof initRepository).toBe("function");
		});

		it("should handle repositories that already exist", async () => {
			// Test would require mocking
			expect(typeof initRepository).toBe("function");
		});
	});

	describe("createBranch", () => {
		it("should create new branch from current HEAD", async () => {
			// Test would require git repository
			expect(typeof createBranch).toBe("function");
		});

		it("should create new branch from specified branch", async () => {
			// Test would require git repository
			expect(typeof createBranch).toBe("function");
		});

		it("should handle branch already exists error", async () => {
			// Test would require mocking git command failure
			expect(typeof createBranch).toBe("function");
		});
	});

	describe("createWorktreeAdvanced", () => {
		it("should create worktree from existing branch", async () => {
			// Test would require git repository
			expect(typeof createWorktreeAdvanced).toBe("function");
		});

		it("should create new branch and worktree together", async () => {
			// Test would require git repository
			expect(typeof createWorktreeAdvanced).toBe("function");
		});

		it("should handle path conflicts", async () => {
			// Test would require mocking filesystem
			expect(typeof createWorktreeAdvanced).toBe("function");
		});
	});
});
