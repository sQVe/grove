import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi } from "vitest";
import { App } from "./App";

// Mock git operations.
vi.mock("../lib/git", () => ({
	listWorktrees: vi.fn(() => Promise.resolve([])),
}));

describe("TUI Launcher", () => {
	it("should render App component without errors", () => {
		const { lastFrame } = render(<App initialWorktrees={[]} />);

		expect(lastFrame()).toContain("Grove");
	});

	it("should handle app initialization with exit handler", () => {
		const onExit = vi.fn();
		const { lastFrame } = render(<App onExit={onExit} />);

		expect(lastFrame()).toContain("Grove");
		expect(onExit).not.toHaveBeenCalled(); // Should not be called during render.
	});

	it("should display empty state correctly", () => {
		const { lastFrame } = render(<App initialWorktrees={[]} />);

		expect(lastFrame()).toContain("No worktrees found");
		expect(lastFrame()).toContain("0 worktrees");
	});
});
