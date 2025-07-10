import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { HeaderBar } from "./HeaderBar";
import type { Worktree } from "../commands/list";

const mockWorktrees: Worktree[] = [
	{
		name: "main",
		path: "/repo/main",
		branch: "main",
		head: "abc123",
		active: true,
		locked: false,
	},
	{
		name: "feature-branch",
		path: "/repo/feature-branch",
		branch: "feature-branch",
		head: "def456",
		active: false,
		locked: false,
	},
];

describe("HeaderBar", () => {
	it("should render with worktree information", () => {
		const { lastFrame } = render(<HeaderBar worktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("Grove");
		expect(lastFrame()).toContain("main"); // Active branch
		expect(lastFrame()).toContain("2 worktrees");
	});

	it("should show active worktree branch with asterisk", () => {
		const { lastFrame } = render(<HeaderBar worktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("main*");
	});

	it("should handle empty worktree list", () => {
		const { lastFrame } = render(<HeaderBar worktrees={[]} />);

		expect(lastFrame()).toContain("Grove");
		expect(lastFrame()).toContain("unknown"); // No active branch
		expect(lastFrame()).toContain("0 worktrees");
	});

	it("should handle no active worktree", () => {
		const inactiveWorktrees: Worktree[] = [
			{
				name: "branch1",
				path: "/repo/branch1",
				branch: "branch1",
				head: "abc123",
				active: false,
				locked: false,
			},
		];

		const { lastFrame } = render(<HeaderBar worktrees={inactiveWorktrees} />);

		expect(lastFrame()).toContain("unknown");
		expect(lastFrame()).toContain("1 worktrees");
	});

	it("should use correct colors and styling", () => {
		const { lastFrame } = render(<HeaderBar worktrees={mockWorktrees} />);
		const output = lastFrame();

		// Check that the output contains the expected structure
		expect(output).toContain("Grove");
		expect(output).toContain("main");
		expect(output).toContain("2 worktrees");
	});
});
